package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	Namespace = "core_namespace"

	SubsystemCommon   = "base"
	SubsystemContract = "contract"
	SubsystemLedger   = "ledger"
	SubsystemState    = "state"
	SubsystemNetwork  = "network"

	LabelBCName      = "bcname"
	LabelMessageType = "message"
	LabelCallMethod  = "method"

	LabelContractModuleName = "contract_module"
	LabelContractName       = "contract_name"
	LabelContractMethod     = "contract_method"
	LabelErrorCode          = "code"

	LabelModule = "module"
	LabelHandle = "handle"
)

var DefBuckets = []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5}

// base
var (
	// 并发请求量
	ConcurrentRequestGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCommon,
			Name:      "concurrent_requests_total",
			Help:      "Total number of concurrent requests.",
		},
		[]string{LabelModule})
	// 字节量
	BytesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCommon,
			Name:      "handle_bytes",
			Help:      "Total size of bytes.",
		},
		[]string{LabelModule, LabelCallMethod, LabelHandle})
	// 函数调用
	CallMethodCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCommon,
			Name:      "call_method_total",
			Help:      "Total number of call method.",
		},
		[]string{LabelModule, LabelCallMethod, LabelErrorCode})
	CallMethodHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCommon,
			Name:      "call_method_seconds",
			Help:      "Histogram of call method cost latency.",
			Buckets:   DefBuckets,
		},
		[]string{LabelModule, LabelCallMethod})
)

// contract
var (
	ContractInvokeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemContract,
			Name:      "invoke_total",
			Help:      "Total number of invoke contract latency.",
		},
		[]string{LabelBCName, LabelContractModuleName, LabelContractName, LabelContractMethod, LabelErrorCode})
	ContractInvokeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemContract,
			Name:      "invoke_seconds",
			Help:      "Histogram of invoke contract latency.",
			Buckets:   DefBuckets,
		},
		[]string{LabelBCName, LabelContractModuleName, LabelContractName, LabelContractMethod})
)

// ledger
var (
	LedgerConfirmTxCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemLedger,
			Name:      "confirmed_tx_total",
			Help:      "Total number of ledger confirmed tx.",
		},
		[]string{LabelBCName})
	LedgerHeightGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SubsystemLedger,
			Name:      "height_total",
			Help:      "Total number of ledger height.",
		},
		[]string{LabelBCName})
	LedgerSwitchBranchCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemLedger,
			Name:      "switch_branch_total",
			Help:      "Total number of ledger switch branch.",
		},
		[]string{LabelBCName})
)

// state
var (
	StateUnconfirmedTxGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SubsystemState,
			Name:      "unconfirmed_tx_gauge",
			Help:      "Total number of miner unconfirmed tx.",
		},
		[]string{LabelBCName})
)

// network
var (
	NetworkMsgSendCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name:      "msg_send_total",
			Help:      "Total number of P2P send message.",
		},
		[]string{LabelBCName, LabelMessageType})
	NetworkMsgSendBytesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name:      "msg_send_bytes",
			Help:      "Total size of P2P send message.",
		},
		[]string{LabelBCName, LabelMessageType})
	NetworkClientHandlingHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name:      "client_handled_seconds",
			Help:      "Histogram of response latency (seconds) of P2P handled.",
			Buckets:   DefBuckets,
		},
		[]string{LabelBCName, LabelMessageType})

	NetworkMsgReceivedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name:      "msg_received_total",
			Help:      "Total number of P2P received message.",
		},
		[]string{LabelBCName, LabelMessageType})
	NetworkMsgReceivedBytesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name:      "msg_received_bytes",
			Help:      "Total size of P2P received message.",
		},
		[]string{LabelBCName, LabelMessageType})
	NetworkServerHandlingHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemNetwork,
			Name:      "server_handled_seconds",
			Help:      "Histogram of response latency (seconds) of P2P handled.",
			Buckets:   DefBuckets,
		},
		[]string{LabelBCName, LabelMessageType})
)

func RegisterMetrics() {
	// base
	prometheus.MustRegister(BytesCounter)
	prometheus.MustRegister(ConcurrentRequestGauge)
	prometheus.MustRegister(CallMethodCounter)
	prometheus.MustRegister(CallMethodHistogram)
	// contract
	prometheus.MustRegister(ContractInvokeCounter)
	prometheus.MustRegister(ContractInvokeHistogram)
	// ledger
	prometheus.MustRegister(LedgerConfirmTxCounter)
	prometheus.MustRegister(LedgerSwitchBranchCounter)
	prometheus.MustRegister(LedgerHeightGauge)
	// state
	prometheus.MustRegister(StateUnconfirmedTxGauge)
	// network
	prometheus.MustRegister(NetworkMsgSendCounter)
	prometheus.MustRegister(NetworkMsgSendBytesCounter)
	prometheus.MustRegister(NetworkClientHandlingHistogram)
	prometheus.MustRegister(NetworkMsgReceivedCounter)
	prometheus.MustRegister(NetworkMsgReceivedBytesCounter)
	prometheus.MustRegister(NetworkServerHandlingHistogram)
}
