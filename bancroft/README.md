# bancroft 
Golang package for estimating a user position using
Bancroft method (Bancroft, 1985) from satellite positions and pseudoranges.

## Usage

To use this package, create a slice of bancroft.SatData structs, each
containing the X, Y, Z coordinates of the satellite and the pseudorange
to the receiver.
The X, Y, Z may be modified according to the travel time, and the pseudorange
may be also modified by known biases such as satellite clock, tropospheric
delay etc.

## Example

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

### Reference:

S. Bancroft, "An Algebraic Solution of the GPS Equations," in IEEE Transactions on Aerospace and Electronic Systems, vol. AES-21, no. 1, pp. 56-59, Jan. 1985, doi: 10.1109/TAES.1985.310538.