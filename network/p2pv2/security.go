package p2pv2

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/sec"
	"github.com/multiformats/go-multiaddr"
	"github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/network/config"
)

// ID is the protocol ID (used when negotiating with multistream)
const ID = "/tls/1.0.0"

// Transport constructs secure communication sessions for a peer.
type Transport struct {
	config *tls.Config

	privKey   crypto.PrivKey
	localPeer peer.ID
}

var _ sec.SecureTransport = &Transport{}

func NewTLS(path, serviceName string) func(key crypto.PrivKey) (*Transport, error) {
	return func(key crypto.PrivKey) (*Transport, error) {
		bs, err := ioutil.ReadFile(filepath.Join(path, "cacert.pem"))
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(bs)
		if !ok {
			return nil, err
		}

		certificate, err := tls.LoadX509KeyPair(filepath.Join(path, "cert.pem"), filepath.Join(path, "private.key"))
		if err != nil {
			return nil, err
		}

		id, err := peer.IDFromPrivateKey(key)
		if err != nil {
			return nil, err
		}

		return &Transport{
			config: &tls.Config{
				ServerName:   serviceName,
				Certificates: []tls.Certificate{certificate},
				RootCAs:      certPool,
				ClientCAs:    certPool,
				ClientAuth:   tls.RequireAndVerifyClientCert,
			},
			privKey:   key,
			localPeer: id,
		}, nil
	}
}

// SecureInbound runs the TLS handshake as a server.
func (t *Transport) SecureInbound(ctx context.Context, insecure net.Conn, p peer.ID) (sec.SecureConn, error) {
	conn := tls.Server(insecure, t.config.Clone())
	if err := conn.Handshake(); err != nil {
		insecure.Close()
		return nil, err
	}

	remotePubKey, err := t.getPeerPubKey(conn)
	if err != nil {
		return nil, err
	}

	return t.setupConn(conn, remotePubKey)
}

// SecureOutbound runs the TLS handshake as a client.
func (t *Transport) SecureOutbound(ctx context.Context, insecure net.Conn, p peer.ID) (sec.SecureConn, error) {
	conn := tls.Client(insecure, t.config.Clone())
	if err := conn.Handshake(); err != nil {
		insecure.Close()
		return nil, err
	}

	remotePubKey, err := t.getPeerPubKey(conn)
	if err != nil {
		return nil, err
	}

	return t.setupConn(conn, remotePubKey)
}

func (t *Transport) getPeerPubKey(conn *tls.Conn) (crypto.PubKey, error) {
	state := conn.ConnectionState()
	if len(state.PeerCertificates) <= 0 {
		return nil, errors.New("expected one certificates in the chain")
	}

	certKeyPub, err := x509.MarshalPKIXPublicKey(state.PeerCertificates[0].PublicKey)
	if err != nil {
		return nil, err
	}

	return crypto.UnmarshalRsaPublicKey(certKeyPub)
}

func (t *Transport) setupConn(tlsConn *tls.Conn, remotePubKey crypto.PubKey) (sec.SecureConn, error) {
	remotePeerID, err := peer.IDFromPublicKey(remotePubKey)
	if err != nil {
		return nil, err
	}

	return &conn{
		Conn:         tlsConn,
		localPeer:    t.localPeer,
		privKey:      t.privKey,
		remotePeer:   remotePeerID,
		remotePubKey: remotePubKey,
	}, nil
}

// conn is SecureConn instance
type conn struct {
	*tls.Conn

	localPeer peer.ID
	privKey   crypto.PrivKey

	remotePeer   peer.ID
	remotePubKey crypto.PubKey
}

var _ sec.SecureConn = &conn{}

func (c *conn) LocalPeer() peer.ID {
	return c.localPeer
}

func (c *conn) LocalPrivateKey() crypto.PrivKey {
	return c.privKey
}

func (c *conn) RemotePeer() peer.ID {
	return c.remotePeer
}

func (c *conn) RemotePublicKey() crypto.PubKey {
	return c.remotePubKey
}

type blankValidator struct{}

func (blankValidator) Validate(_ string, _ []byte) error        { return nil }
func (blankValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

func Key(account string) string {
	return fmt.Sprintf("/%s/account/%s", base.Namespace, account)
}

func GenAccountKey(account string) string {
	return fmt.Sprintf("/%s/account/%s", base.Namespace, account)
}

func GenPeerIDKey(id peer.ID) string {
	return fmt.Sprintf("/%s/id/%s", base.Namespace, id)
}

// GetPeerIDByAddress return peer ID corresponding to peerAddr
func GetPeerIDByAddress(peerAddr string) (peer.ID, error) {
	addr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return "", err
	}
	peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return "", err
	}
	return peerinfo.ID, nil
}

// GetPemKeyPairFromPath get xuper pem private key from file path
func GetPemKeyPairFromPath(path string) (crypto.PrivKey, error) {
	if len(path) == 0 {
		path = config.DefaultNetKeyPath
	}

	keyFile, err := ioutil.ReadFile(filepath.Join(path, "private.key"))
	if err != nil {
		return nil, err
	}

	keyBlock, _ := pem.Decode(keyFile)
	return crypto.UnmarshalRsaPrivateKey(keyBlock.Bytes)
}

// GeneratePemKeyFromNetKey get pem format private key from net private key
func GeneratePemKeyFromNetKey(path string) error {
	privKey, err := GetKeyPairFromPath(path)
	if err != nil {
		return err
	}

	bytes, err := privKey.Raw()
	if err != nil {
		return err
	}

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bytes,
	}
	return ioutil.WriteFile(filepath.Join(path, "private.key"), pem.EncodeToMemory(block), 0700)
}

// GetKeyPairFromPath get xuper net key from file path
func GetKeyPairFromPath(path string) (crypto.PrivKey, error) {
	if len(path) == 0 {
		path = config.DefaultNetKeyPath
	}

	data, err := ioutil.ReadFile(filepath.Join(path, "net_private.key"))
	if err != nil {
		return nil, err
	}

	privData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPrivateKey(privData)
}
