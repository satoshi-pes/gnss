package antex

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	mscanner "github.com/satoshi-pes/modscanner"
)

// crinex logger
var logger = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)

// antenna stores contents in a ANTEX file
type antenna struct {
	// Antenna type
	Type string

	// S1:
	// - Serial number for receiver antenna
	// - Satellite ID (e.g., "G01") for satellite antenna
	S1 string

	// S2: blank for recant, optional for satant
	// -Satellite ID for satellite antenna
	S2 string // optional

	// S3: blank for recant, optional for satant
	// - COSPARID ("YYYY-XXXA") for satellite antenna
	S3 string // optional

	// Antenna phase center variation
	PCV []pcv

	isSatelliteAntenna bool

	// optional records
	ValidFrom, ValidUntil time.Time
	SinexCode             string
}

func (a *antenna) IsSatAnt() bool {
	return a.isSatelliteAntenna
}

type pco struct {
	N, E, U float64
}

type pcv struct {
	Sys        byte
	Freq       int
	Zen1, Zen2 float64
	Dzen       float64
	Dazi       float64
	Nazi, Nzen int

	PCO pco

	// PCV
	Vnonaz []float64   // non-azimuth dependent PCV
	Vaz    [][]float64 // azimuth dependent PCV
	Azs    []float64
}

// ReadAntexFile reads and returns the contents of a ANTEX file as antenna struct.
func ReadAntexFile(antexFile string) (ver string, satSys, pcvType byte, refAnt string, antennas []antenna) {
	f, err := os.Open(antexFile)
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	//s := bufio.NewScanner(f)
	s := mscanner.NewScanner(f)

	antexVer, satSys, pcvType, refAnt, h := ScanHeader(s)
	_ = h
	ant := ReadAntexData(s)

	return antexVer, satSys, pcvType, refAnt, ant
}

// scanHeader parses the header
func ScanHeader(s *mscanner.Scanner) (antexVer string, satSys, pcvType byte, refAnt string, h []byte) {
	for s.Scan() {
		buf := s.Text()
		if len(buf) < 61 {
			// no header label found, and read as a comment
			logger.Printf("warning: no header label found: s='%s'\n", buf)
			buf = fmt.Sprintf("%-60sCOMMENT", buf)
		}

		h = append(h, []byte(buf)...)
		h = append(h, byte('\n'))

		switch {
		case strings.HasPrefix(buf[60:], "ANTEX VERSION / SYST"):
			antexVer = strings.TrimSpace(buf[:8])
			satSys = byte(buf[20])
		case strings.HasPrefix(buf[60:], "PCV TYPE / REFANT"):
			pcvType = byte(buf[0])
			refAnt = strings.TrimSpace(buf[20:40])
		case strings.HasPrefix(buf[60:], "COMMENT"):
			// do nothing
			continue
		case strings.HasPrefix(buf[60:], "END OF HEADER"):
			return
		default:
			// do nothing
			continue
		}
	}

	// "END OF HEADER" not found

	return
}

// scanHeader parses the header
func ReadAntexData(s *mscanner.Scanner) (ant []antenna) {
	for s.Scan() {
		buf := s.Text()
		if len(buf) < 61 {
			logger.Printf("warning: no antex data found: s='%s'\n", buf)
		}

		if strings.HasPrefix(buf[60:], "START OF ANTENNA") {
			a, e := scanOneAntenna(s)
			if e != nil {
				logger.Printf("error: %v\n", e)
			}
			ant = append(ant, a)
		}
	}

	return
}

func scanOneAntenna(s *mscanner.Scanner) (ant antenna, err error) {
	var (
		nf                     int
		dazi, zen1, zen2, dzen float64
		e                      error
	)
	ant = antenna{}
	ant.PCV = make([]pcv, 0)

	/*
	*                                                            START OF ANTENNA
	*BLOCK IIA           G01                 G032      1992-079A TYPE / SERIAL NO
	*                                             0    29-JAN-17 METH / BY / # / DATE
	*     0.0                                                    DAZI
	*     0.0  17.0   1.0                                        ZEN1 / ZEN2 / DZEN
	*     2                                                      # OF FREQUENCIES
	*  1992    11    22     0     0    0.0000000                 VALID FROM
	*  2008    10    16    23    59   59.9999999                 VALID UNTIL
	*IGS14_2247                                                  SINEX CODE
	*   G01                                                      START OF FREQUENCY
	*    279.00      0.00   2319.50                              NORTH / EAST / UP
	*   NOAZI   -0.80   -0.90   -0.90   -0.80   -0.40    0.20    0.80    1.30    1.40    1.20    0.70    0.00   -0.40   -0.70   -0.90   -0.90   -0.90   -0.90
	*   G01                                                      END OF FREQUENCY
	*   G02                                                      START OF FREQUENCY
	*    279.00      0.00   2319.50                              NORTH / EAST / UP
	*   NOAZI   -0.80   -0.90   -0.90   -0.80   -0.40    0.20    0.80    1.30    1.40    1.20    0.70    0.00   -0.40   -0.70   -0.90   -0.90   -0.90   -0.90
	*   G02                                                      END OF FREQUENCY
	*                                                            END OF ANTENNA
	 */

	// read
	for s.Scan() {
		e = nil

		buf := s.Text()
		if len(buf) < 61 {
			logger.Printf("warning: no label found: s='%s'\n", buf)
			break
		}

		label := strings.TrimRight(buf[60:], " ")

		switch label {
		case "COMMENT":
			continue
		case "TYPE / SERIAL NO":
			ant.Type = strings.TrimSpace(buf[:20])
			ant.S1 = strings.TrimSpace(buf[20:40])
			ant.S2 = strings.TrimSpace(buf[40:50])
			ant.S3 = strings.TrimSpace(buf[50:60])
		case "METH / BY / # / DATE":
		case "DAZI":
			dazi, e = strconv.ParseFloat(strings.TrimSpace(buf[:60]), 64)

		case "ZEN1 / ZEN2 / DZEN":
			sep := strings.Fields(buf[:60])
			if len(sep) < 3 {
				break
			}
			var e1, e2, e3 error
			zen1, e1 = strconv.ParseFloat(sep[0], 64)
			zen2, e2 = strconv.ParseFloat(sep[1], 64)
			dzen, e3 = strconv.ParseFloat(sep[2], 64)
			for _, ee := range []error{e1, e2, e3} {
				if ee != nil {
					e = ee
					break
				}
			}
		case "# OF FREQUENCIES":
			if nf, e = strconv.Atoi(strings.TrimSpace(buf[:60])); e != nil {
				_ = nf
			}
		case "VALID FROM":
			ant.ValidFrom, e = parseDate(buf[:60])
		case "VALID UNTIL":
			ant.ValidUntil, e = parseDate(buf[:60])
		case "SINEX CODE":
			ant.SinexCode = strings.TrimSpace(buf[:60])
		case "START OF FREQUENCY":
			p, e1 := parseOneFreq(s, buf, dazi, zen1, zen2, dzen)
			if e1 != nil {
				//logger.Printf("error found in reading pcv data for '%s', err='%v'\n", ant.Type, e1)
				e = fmt.Errorf("error in pcv data for '%s', err='%w'\n", ant.Type, e1)
				break
			}

			p.Dazi = dazi
			p.Zen1 = zen1
			p.Zen2 = zen2
			p.Dzen = dzen
			ant.PCV = append(ant.PCV, p)

		case "END OF ANTENNA":
			// success return
			return
		}

		// break loop in error
		if e != nil {
			break
		}
	}

	// error: end label not found
	if e == nil {
		err = fmt.Errorf("End of freq label not found")
		return
	}

	// advance to the end of freq label
	if e != nil {
		for s.Scan() {
			buf := s.Text()
			if len(buf) > 60 && strings.HasPrefix(buf[60:], "END OF ANTENNA") {
				break
			}
		}
		err = e
	}

	return
}

func parseOneFreq(s *mscanner.Scanner, buf string, dazi, zen1, zen2, dzen float64) (p pcv, err error) {
	// nextLine returns the next line skipping "COMMENT"
	nextLine := func(s *mscanner.Scanner) (line string) {
		for s.Scan() {
			line = s.Text()
			if !strings.HasSuffix(strings.TrimRight(buf, " "), "COMMENT") {
				return
			}
		}
		return ""
	}

	// freq record
	p.Sys = buf[3]
	p.Freq, err = strconv.Atoi(buf[4:6])
	if err != nil {
		return
	}

	buf = nextLine(s)
	if strings.HasPrefix(buf[60:], "NORTH / EAST / UP") {
		// pco neu
		sep := strings.Fields(buf[:60])
		if len(sep) < 3 {
			// error return advancing position to the END OF FREQUENCY
			for s.Scan() {
				buf = s.Text()
				if strings.HasPrefix(buf[60:], "END OF FREQUENCY") {
					return
				}
			}
		}

		n, e1 := strconv.ParseFloat(sep[0], 64)
		e, e2 := strconv.ParseFloat(sep[1], 64)
		u, e3 := strconv.ParseFloat(sep[2], 64)

		for _, ee := range []error{e1, e2, e3} {
			if ee != nil {
				return p, ee
			}
		}

		// ok
		p.PCO.N, p.PCO.E, p.PCO.U = n, e, u
	}

	// numbers of azimuth, zenith angels
	var nazi, nzen int
	if dazi == 0 {
		nazi = 0
	} else {
		nazi = int(360./dazi) + 1
	}
	nzen = int((zen2-zen1)/dzen) + 1

	p.Nzen, p.Nazi = nzen, nazi

	// allocation
	p.Vaz = make([][]float64, nazi)
	p.Azs = make([]float64, nazi)

	// scan noazi
	buf = nextLine(s)

	// PCV for NOAZI
	az, vals, e := parseOneAzi(buf)
	switch {
	case e != nil:
		return p, fmt.Errorf("line=%d, err='%w'", s.LineNumber(), e)
	case az != "NOAZI":
		return p, fmt.Errorf("NOAZI not found: line=%d, buf='%s'", s.LineNumber(), buf)
	case len(vals) != nzen:
		return p, fmt.Errorf("invalid values: line=%d, nzen=%d, len=%d, buf='%s'", s.LineNumber(), nzen, len(vals), buf)
	}
	p.Vnonaz = vals

	// start to read pcv
	for i := 0; i < nazi; i++ {
		buf = nextLine(s)
		az, vals, e = parseOneAzi(buf)

		// error check
		switch {
		case e != nil:
			return p, fmt.Errorf("line=%d, err='%w'", s.LineNumber(), e)
		case az == "NOAZI":
			return p, fmt.Errorf("NOAZI found at azimuth dependent values: line=%d, buf='%s'", s.LineNumber(), buf)
		case len(vals) != nzen:
			return p, fmt.Errorf("invalid values: line=%d, nzen=%d, len=%d, buf='%s'", s.LineNumber(), nzen, len(vals), buf)
		}

		p.Vaz[i] = vals
		p.Azs[i], e = strconv.ParseFloat(az, 64)
		if e != nil {
			err = fmt.Errorf("invalid az found: line=%d, az='%s'", s.LineNumber(), az)
			return
		}
	}

	// advance a line
	buf = nextLine(s)

	// check the end of frequency
	if len(buf) < 61 || !strings.HasPrefix(buf[60:], "END OF FREQUENCY") {
		// skip to the next "END OF FREQUENCY"
		for s.Scan() {
			buf = s.Text()
			if len(buf) > 60 || strings.HasPrefix(buf[60:], "END OF FREQUENCY") {
				return p, fmt.Errorf("invalid 'END OF FREQUENCY' position: line=%d", s.LineNumber())
			}
		}
		return p, fmt.Errorf("'END OF FREQUENCY' not found: line=%d", s.LineNumber())
	}

	// success return
	return
}

func parseOneAzi(s string) (az string, v []float64, e error) {
	sep := strings.Fields(s)

	if len(sep) == 0 {
		e = fmt.Errorf("no phase variation found: s='%s'", s)
		return
	}

	// phase center variation for an azimuth
	// sep: az, v0, v1, ...,vn
	az = sep[0] // note: this is still a string at this point
	v = make([]float64, len(sep)-1)

	for i := 0; i < len(sep)-1; i++ {
		v[i], e = strconv.ParseFloat(sep[i+1], 64)
		if e != nil {
			e = fmt.Errorf("failed to parse val: '%s', err='%w'", sep[i+1], e)
			return
		}
	}

	return az, v, nil
}

func parseDate(s string) (t time.Time, e error) {
	//  2016    12     9     0     0    0.0000000                 VALID FROM
	//  2017     1     3    23    59   59.9999999                 VALID UNTIL
	const timeForm = "2006     1     2    15     4    5        "
	return time.Parse(timeForm, strings.TrimSpace(s))
}
