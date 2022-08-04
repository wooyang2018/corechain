/*
Copyright Baidu Inc. All Rights Reserved.

<jingbo@baidu.com>
*/

package complex

import (
	"crypto/elliptic"
	"errors"
	"log"
	"math/big"

	"github.com/wooyang2018/corechain/crypto/common/secret/polynomial"
)

var (
	EmptyMessageError = errors.New("The message to be signed should not be empty")
)

// Shamir's Secret Sharing algorithm, can be considered as:
// A way to split a secret to W shares, the secret can only be retrieved if more than T(T <= W) shares are combined together.
//
// This is the retrieve process:
// 1. Decode each share i.e. the byte slice to a (x, y) pair
// 2. Use lagrange interpolation formula, take the (x, y) pairs as input points to compute a polynomial f(x)
//		 which is able to match all the given points.
// 3. Give x = 0, then the secret number S can be computed
// 4. Now decode number S, then the secret is retrieved
func ComplexSecretRetrieve2(shares map[int]*big.Int, curve elliptic.Curve) ([]byte, error) {
	secretInt := lagrangeInterpolate2(shares, 0, curve)

	secret := secretInt.Bytes()

	return secret, nil
}

// Lagrange Polynomial Interpolation Formula
func lagrangeInterpolate2(points map[int]*big.Int, x int, curve elliptic.Curve) *big.Int {
	log.Printf("The points is: %v", points)

	// 通过这些坐标点来恢复出多项式
	polynomialClient := polynomial.New(curve.Params().N)
	result := polynomialClient.GetPolynomialByPoints(points)

	// 秘密就是常数项
	secret := result[len(result)-1]

	log.Printf("The coefficients of the polynomial is: %v", result)
	return secret
}

// Shamir's Secret Sharing algorithm, can be considered as:
// A way to split a secret to W shares, the secret can only be retrieved if more than T(T <= W) shares are combined together.
//
// This is the retrieve process:
// 1. Decode each share i.e. the byte slice to a (x, y) pair
// 2. Use lagrange interpolation formula, take the (x, y) pairs as input points to compute a polynomial f(x)
//		 which is able to match all the given points.
// 3. Give x = 0, then the secret number S can be computed
// 4. Now decode number S, then the secret is retrieved
func ComplexSecretRetrieve(shares map[int]*big.Int) ([]byte, error) {
	secretInt := lagrangeInterpolate(shares, 0)

	secret := secretInt.Bytes()

	return secret, nil
}

// Lagrange Polynomial Interpolation Formula
func lagrangeInterpolate(points map[int]*big.Int, x int) *big.Int {
	log.Printf("The points is: %v", points)

	// 通过这些坐标点来恢复出多项式
	result := polynomial.GetPolynomialByPoints(points)

	// 秘密就是常数项
	secret := result[len(result)-1]

	log.Printf("The coefficients of the polynomial is: %v", result)
	return secret
}
