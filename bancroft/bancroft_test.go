package bancroft

import (
	"fmt"
	"log"
	"testing"
)

func ExampleCalcPos() {
	satDatas := []SatData{
		{X: -12005459.353, Y: 22848755.674, Z: 5796967.796, PR: 21103816.114230197},
		{X: -26293588.245, Y: -625514.504, Z: -4190661.860, PR: 24293188.495801315},
		{X: -6559774.102, Y: 22208149.128, Z: 13685049.829, PR: 21325719.850603405},
		{X: 3709341.143, Y: 24439380.765, Z: 9629909.454, PR: 22938884.45379082},
	}

	x, y, z, dt, err := CalcPos(satDatas)
	if err != nil {
		log.Fatalf("Failed to calculate position: %v", err)
	}

	fmt.Printf("pos: x=%.3f, y=%.3f, z=%.3f, dt=%e\n", x, y, z, dt)
	// Output: pos: x=-3721817.095, y=3545589.644, z=3763571.160, dt=-4.271441e-07
}

// TestCalcPos calls bancroft.CalcPos with a prepared data
func TestCalcPos(t *testing.T) {
	// test data
	satDatas := make([]SatData, len(satPosData))
	for i, sp := range satPosData {
		satDatas[i].X = sp.X * 1000. // km -> m
		satDatas[i].Y = sp.Y * 1000. // km -> m
		satDatas[i].Z = sp.Z * 1000. // km -> m
		satDatas[i].PR = rangeData[i] + sp.C*0.000001*LightVelocity
	}

	// expected solution
	x0 := -3.7217662230820884e+06
	y0 := 3.5454831981535875e+06
	z0 := 3.763601929807223e+06
	dt0 := -1.8788904835860208e-07

	// test
	x, y, z, dt, err := CalcPos(satDatas)

	if x != x0 || y != y0 || z != z0 || dt != dt0 || err != nil {
		t.Errorf("\nget (x, y, z, dt) = %f, %f, %f, %e\nwant(x, y, z, dt) = %f, %f, %f, %e\nerr: %v", x, y, z, dt, x0, y0, z0, dt0, err)
	}

	// compare site position in the RINEX header
	// run "go test -v" to show
	var msg string
	msg += "testing using data of GEONET site 0255 (KOMATSU)\n"
	msg += "RINEX: 02551960.24o\n"
	msg += "epoch: 2024-07-14 00:00:00.000\n"
	msg += "site position (RINEX header):  -3721695.1985  3545492.6126  3763541.7139\n"
	msg += fmt.Sprintf("site position (by test)     :  %.4f  %.4f  %.4f\n", x, y, z)
	msg += fmt.Sprintf("site position (want)        :  %.4f  %.4f  %.4f\n", x0, y0, z0)
	t.Logf(msg)
}

// This stores pseudoranges observed at the GEONET site "0255" (KOMATSU station).
//
// used data: 02551960.24o
// timetag  : 2024-07-14 00:00:00.0000000
// satellite: GPS (G14, G04, G22, G06, G17, G03, G21, G19, G02)
// obs code : C1C
//
// The data was obtained from Geospatial Information Authority Website:
// https://terras.gsi.go.jp
var rangeData = []float64{
	20969460.336, // G14
	24172308.906, // G04
	21337486.820, // G22
	22889983.813, // G06
	20931837.352, // G17
	20333361.633, // G03
	24731230.883, // G21
	22113357.180, // G19
	23285127.484, // G02
}

// satPosData is a test data storing satellite precise ephemeris obrained from
// IGS rapid orbit of "igr23230.sp3".
//
// used data: igr23230.sp3
// timetag  : 2024-07-14 00:00:00.000000
// satellite: GPS (G14, G04, G22, G06, G17, G03, G21, G19, G02)
// parameter: x(km), y(km), z(km), c(microsec)
var satPosData = []satPos{
	{-12005.459353, 22848.755674, 5796.967796, 448.162636},   // G14
	{-26293.588245, -625.514504, -4190.661860, 403.210910},   // G04
	{-6559.774102, 22208.149128, 13685.049829, -39.250385},   // G22
	{3709.341143, 24439.380765, 9629.909454, 163.114980},     // G06
	{-7463.615200, 13958.181703, 21770.913274, 678.028776},   // G17
	{-19010.942887, 7137.091598, 16892.674725, 456.857028},   // G03
	{-19057.263732, -12281.672714, 15058.503648, 109.105768}, // G21
	{3212.240631, 15559.603142, 21180.943665, 510.053160},    // G19
	{-18350.488725, -6421.169947, 18706.745770, -399.560198}, // G02
}

type satPos struct {
	X, Y, Z, C float64
}
