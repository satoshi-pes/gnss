/*
package bancroft provides functionalyti for estimating a user position using
Bancroft method (Bancroft, 1985) from satellite positions and pseudoranges.

Usage:

To use this package, create a slice of bancroft.SatData structs, each
containing the X, Y, Z coordinates of the satellite and the pseudorange
to the receiver.
The X, Y, Z may be modified according to the travel time, and the pseudorange
may be also modified by known biases such as satellite clock, tropospheric
delay etc.

Example:

	satDatas := []SatData{
		{X: -12005459.353, Y: 22848755.674, Z:  5796967.796, PR: 21103816.114230197},
		{X: -26293588.245, Y:  -625514.504, Z: -4190661.860, PR: 24293188.495801315},
		{X:  -6559774.102, Y: 22208149.128, Z: 13685049.829, PR: 21325719.850603405},
		{X:   3709341.143, Y: 24439380.765, Z:  9629909.454, PR: 22938884.45379082},
	}

	x, y, z, dt, err := CalcPos(satDatas)
	if err != nil {
		log.Fatalf("Failed to calculate position: %v", err)
	}

	fmt.Printf("pos: x=%.3f, y=%.3f, z=%.3f, dt=%e\n", x, y, z, dt)

Reference:

S. Bancroft, "An Algebraic Solution of the GPS Equations," in IEEE Transactions on Aerospace and Electronic Systems, vol. AES-21, no. 1, pp. 56-59, Jan. 1985, doi: 10.1109/TAES.1985.310538.
*/
package bancroft

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// Speed of light (m/s)
const LightVelocity = 299792458.

// SatData defines the input data for Bancroft().
// X, Y, Z (m) are the satellite position, and PR is the pseudorange (m).
// X, Y, Z may be modified by the traveltime (PR/C), and PR could be corrected
// by known biases such as tropospheric delay before the call of Bancroft().
type SatData struct {
	X, Y, Z float64 // satellite position (m)
	PR      float64 // pseudorange (m)
}

// CalcPos solves the GNSS equation using Bancroft method (Bancroft, 1985).
func CalcPos(satDatas []SatData) (x, y, z, dt float64, err error) {
	// make B matrix and i0, r vectors
	//
	// A  = (a1, a2, ..., an)'  (eq.5)
	// i0 = (1, 1, ..., 1)'     (eq.6)
	// r  = (r1, r2, ..., rn)'  (eq.7)
	// where ri = <ai,ai>/2
	//
	// B: the generalized inverse of A:
	//    B = (A'A)^-1 A'       (eq.9)
	//
	B, r, i0, err := constructBancroftMatrices(satDatas)
	if err != nil {
		return 0., 0., 0., 0., err
	}

	// solve the quadratic equation by Bancroft for lambda:
	// <u,u>lam^2 + 2(<u,v>-1)lam + <v,v> = 0   (eq.15)
	var u, v mat.VecDense
	u.MulVec(B, i0) // (eq.10)
	v.MulVec(B, r)  // (eq.11)

	lam1, lam2, err := solveBancroftQuadraticEq(u, v)
	if err != nil {
		return 0., 0., 0., 0., err
	}

	// (eq.16)
	// possible two solutions
	s1 := make([]float64, 4)
	s2 := make([]float64, 4)

	for i := range 4 {
		s1[i] = lam1*u.AtVec(i) + v.AtVec(i)
		s2[i] = lam2*u.AtVec(i) + v.AtVec(i)
	}

	// test two possible solutions.
	// the solution closer to the Earth's surface is adopted as the true solution.
	EarthRadius := 6378000. // Earth's radius (m)
	res1 := math.Abs(EarthRadius - math.Sqrt(sqr(s1[0])+sqr(s1[1])+sqr(s1[2])))
	res2 := math.Abs(EarthRadius - math.Sqrt(sqr(s2[0])+sqr(s2[1])+sqr(s2[2])))

	x, y, z, dt = s1[0], s1[1], s1[2], s1[3]/LightVelocity
	if res2 < res1 {
		x, y, z, dt = s2[0], s2[1], s2[2], s2[3]/LightVelocity
	}

	return x, y, z, dt, nil
}

func solveBancroftQuadraticEq(u, v mat.VecDense) (lam1, lam2 float64, err error) {
	// (eq.12)
	E, err := minkowski4D(u, u)
	if err != nil {
		return 0., 0., err
	}

	// (eq.13)
	uv, err := minkowski4D(u, v)
	if err != nil {
		return 0., 0., err
	}
	F := uv - 1.

	// (eq.14)
	G, err := minkowski4D(v, v)
	if err != nil {
		return 0., 0., err
	}

	// (eq.15)
	// solve the quadratic equation Ex^2 + 2Fx + G = 0
	a, b, c := E, F, G
	lam1 = (-b + math.Sqrt(b*b-a*c)) / a // solution1
	lam2 = (-b - math.Sqrt(b*b-a*c)) / a // solution2

	return lam1, lam2, nil
}

// Matrices constructs matrices of eqs (6), (7), (9) defined in Bancroft (1985).
//
// A  = (a1, a2, ..., an)'  (eq.5)
// i0 = (1, 1, ..., 1)'     (eq.6)
// r  = (r1, r2, ..., rn)'  (eq.7)
// where ri = <ai,ai>/2
//
// B: the generalized inverse of A:
//
//	B = (A'A)^-1 A'         (eq.9)
func constructBancroftMatrices(satDatas []SatData) (B *mat.Dense, r, i0 *mat.VecDense, err error) {
	n := len(satDatas)
	if n < 4 {
		err = fmt.Errorf("not enough satellite")
		return
	}

	// observation equation matrix
	A := mat.NewDense(n, 4, nil)

	// matrices to be returned
	B = mat.NewDense(4, n, nil)
	r = mat.NewVecDense(n, nil)
	i0 = mat.NewVecDense(n, nil)

	for i, s := range satDatas {
		A.Set(i, 0, s.X)
		A.Set(i, 1, s.Y)
		A.Set(i, 2, s.Z)
		A.Set(i, 3, s.PR)

		r.SetVec(i, 0.5*(sqr(s.X)+sqr(s.Y)+sqr(s.Z)-sqr(s.PR)))
		i0.SetVec(i, 1.)
	}

	// inverse of A
	switch {
	case n == 4:
		err = B.Inverse(A)
		if err != nil {
			return
		}
	case n > 4:
		B, err = generalizedInverse(A)
		if err != nil {
			return
		}
	}

	return B, r, i0, nil
}

// Minkowski4D returns following result for two 4-dimensional vectors.
// <a,b> = a1*b1 + a2*b2 + a3*b3 - a4*b4
//
// Note that above operation is similar to the spacetime interval for
// the coordinate system (x, y, z, ct):
// ds^2 = dx^2 + dy^2 + dz^2 - (cdt)^2
func minkowski4D(a, b mat.VecDense) (float64, error) {
	return calcMinkowski4D(a.RawVector().Data, b.RawVector().Data)
}

func calcMinkowski4D(a, b []float64) (v float64, err error) {
	if len(a) != 4 || len(b) != 4 {
		return v, fmt.Errorf("invalid vector size")
	}
	v = a[0]*b[0] + a[1]*b[1] + a[2]*b[2] - a[3]*b[3]
	return v, nil
}

// generalizedInverse returns the generalized inverse of the given matrix A.
func generalizedInverse(A *mat.Dense) (*mat.Dense, error) {
	var AtA, AtAi mat.Dense
	var AI mat.Dense

	AtA.Mul(A.T(), A)         // A^t*A
	err := AtAi.Inverse(&AtA) // (A^t*A)^-1
	if err != nil {
		return &AI, err
	}

	// generalized inverse
	AI.Mul(&AtAi, A.T()) // (A^t*A)^-1 * A^t

	return &AI, nil
}

func sqr(x float64) float64 {
	return x * x
}
