package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 10 {
		fmt.Fprintln(os.Stderr, "USAGE:", os.Args[0], "IMAGE_FILE CELL_SIZE X0 Y0 R_TH G_TH B_TH A_TH SAMPLES")
		return
	}

	infile, err := os.Open(os.Args[1])

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	defer infile.Close()

	cellsize, err := strconv.Atoi(os.Args[2])

	if err != nil {
		fmt.Fprintln(os.Stderr, "Reading cell size:", err)
		return
	}

	x0, err := strconv.Atoi(os.Args[3])

	if err != nil {
		fmt.Fprintln(os.Stderr, "Reading x0:", err)
		return
	}

	y0, err := strconv.Atoi(os.Args[4])

	if err != nil {
		fmt.Fprintln(os.Stderr, "Reading y0:", err)
		return
	}

	rth, err := strconv.Atoi(os.Args[5])

	if err != nil {
		fmt.Fprintln(os.Stderr, "Reading red threshold:", err)
		return
	}

	gth, err := strconv.Atoi(os.Args[6])

	if err != nil {
		fmt.Fprintln(os.Stderr, "Reading green threshold:", err)
		return
	}

	bth, err := strconv.Atoi(os.Args[7])

	if err != nil {
		fmt.Fprintln(os.Stderr, "Reading blue threshold:", err)
		return
	}

	ath, err := strconv.Atoi(os.Args[8])

	if err != nil {
		fmt.Fprintln(os.Stderr, "Reading alpha threshold:", err)
		return
	}

	samples, err := strconv.Atoi(os.Args[9])

	if err != nil {
		fmt.Fprintln(os.Stderr, "Reading samples:", err)
		return
	}

	img, _, err := image.Decode(infile)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	rect := img.Bounds()
	imgsize := rect.Max.Sub(rect.Min)
	gridsize := imgsize.Div(cellsize)
	sampleStep := cellsize / samples
	fmt.Fprintln(os.Stderr, "Bounds:", img.Bounds())
	fmt.Fprintln(os.Stderr, "Cell size:", cellsize)
	fmt.Fprintln(os.Stderr, "Image size:", imgsize)
	fmt.Fprintln(os.Stderr, "Grid size:", gridsize)
	fmt.Fprintln(os.Stderr, "Sample step:", sampleStep)

	samplePoints := make([]image.Point, 0, samples*samples)

	for i := 0; i < samples; i++ {
		for j := 0; j < samples; j++ {
			samplePoints = append(samplePoints, image.Point{i * sampleStep, j * sampleStep})
		}
	}

	fmt.Print("insert into nodes (x, y, yield) values ")

	first := true

	for i := 0; i < gridsize.Y; i++ {
		for j := 0; j < gridsize.X; j++ {
			for _, s := range samplePoints {
				r, g, b, a := img.At(cellsize*j+s.X, cellsize*i+s.Y).RGBA()
				r >>= 8
				g >>= 8
				b >>= 8
				a >>= 8

				if r < uint32(rth) || g < uint32(gth) || b < uint32(bth) || a < uint32(ath) {
					if !first {
						fmt.Print(",")
					}

					fmt.Println("(", j+x0, ",", i+y0, ", 0)")
					first = false

					break
				}
			}
		}
	}

	fmt.Println(";")
}
