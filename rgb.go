package lstdp

import (
	"log"
	
	"github.com/daviddengcn/go-vision"
)

type SegmentOpt struct {
	T int
	MaxAdjustX int
	Rx, Ry int
}

type RGBOption struct {
	Option
	Segment SegmentOpt
}

type lineSegment struct {
	Y      int
	XStart int
	Len    int
}

func abs(vl int) int {
	if vl < 0 {
		return -vl
	}
	return vl
}

func sumabs3(a, b, c int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	if c < 0 {
		c = -c
	}
	return a + b + c
}

func rgbDiff(a vision.RGB, b vision.RGB) int {
	return sumabs3(int(a[0]) - int(b[0]), int(a[1]) - int(b[1]), int(a[2]) - int(b[2]))
}

func reverse(b byte) byte {
	j := byte(0x80)
	res := byte(0)
	for i := 1; i < 0x100; i <<= 1 {
		if int(b) & i != 0 {
			res |= j
		}
		j >>= 1
	}
	return res
}

func rgbSegment(img vision.RGBImage, opt SegmentOpt) (idxImage vision.IntGrayImage) {
	var marks vision.GrayImage
	marks.Resize(img.Size)
	marks.Fill(0)
	for y := 0; y < img.Height; y++ {
		start := y*img.Width
		
		// initial marks
		marks.Pixels[start] = 255
		
		mn, mx := img.Pixels[start], img.Pixels[start]
		for x := 1; x < img.Width; x++ {
			p := img.Pixels[start + x]
			
			toMark := false
			for i := 0; i < 3; i ++ {
				if p[i] < mn[i] {
					mn[i] = p[i]
				} else if p[i] > mx[i] {
					mx[i] = p[i]
				}
				if int(mx[i] - mn[i]) > opt.T {
					toMark = true
					break
				}
			}
			if toMark {
				marks.Pixels[start + x] = 255
				mn, mx = p, p
			}
		}
		
		// repositioning
		for x := 1; x < img.Width; x++ {
			if marks.Pixels[start + x] == 0 {
				continue
			}
			
			maxDiff := rgbDiff(img.Pixels[start + x], img.Pixels[start + x - 1])
			maxX := x
			for x1 := x - 1; x1 >= x - opt.MaxAdjustX && x1 > 0; x1-- {
				if marks.Pixels[start + x1] != 0 {
					break
				}
				diff := rgbDiff(img.Pixels[start + x1], img.Pixels[start + x1 - 1])
				if diff > maxDiff {
					maxDiff, maxX = diff, x1
				}
			}
			for x1 := x + 1; x1 <= x + opt.MaxAdjustX && x1 < img.Width; x1++ {
				if marks.Pixels[start + x1] != 0 {
					break
				}
				diff := rgbDiff(img.Pixels[start + x1], img.Pixels[start + x1 - 1])
				if diff > maxDiff {
					maxDiff, maxX = diff, x1
				}
			}
			
			if maxX != x {
				marks.Pixels[start + x], marks.Pixels[start + maxX] = 0, 255
			}
		}
	}
	
	// remove isolated marks
	for y := 0; y < marks.Height; y++ {
		for x := 1; x < marks.Width; x++ {
			offs := y*marks.Width + x
			if marks.Pixels[offs] == 0 {
				continue
			}
			
			if !func() bool {
				for dy := -opt.Ry; dy <= opt.Ry; dy++ {
					y1 := y + dy
					if y1 < 0 {
						continue
					}
					if y1 >= marks.Height {
						break
					}
					
					for dx := -opt.Rx; dx <= opt.Rx; dx++ {
						x1 := x + dx
						if x1 < 0 {
							continue
						}
						if x1 >= marks.Width {
							break
						}
						
						if dx != 0 || dy != 0 {
							if marks.Pixels[y1*marks.Width + x1] != 0 {
								return true
							}
						}
					}
				}
				return false
			}() {
				marks.Pixels[offs] = 0
			}
		}
	}
	
	idxImage.Resize(marks.Size)
	l := idxImage.Area()
	segmIdx := -1
	for i := 0; i < l; i ++ {
		if marks.Pixels[i] != 0 {
			segmIdx++
		}
		idxImage.Pixels[i] = segmIdx
	}
	
	err := vision.SaveImageAsPng(marks.AsImage(), "/tmp/marks.png")
	log.Printf("%v", err)
	
	/*
	var rgbClImage vision.RGBImage
	rgbClImage.Resize(idxImage.Size)
	for i := 0; i < l; i++ {
		idx := idxImage.Pixels[i]
		rgbClImage.Pixels[i][0] = reverse(byte(idx % 256))
		rgbClImage.Pixels[i][1] = reverse(byte((idx/2) % 256))
		rgbClImage.Pixels[i][2] = reverse(byte((idx/4) % 256))
	}
	err = vision.SaveImageAsPng(rgbClImage.AsImage(), "/tmp/segms.png")
	log.Printf("%v", err)
	*/
	
	return
}

func minmax(a, b, c byte) (mn, mx byte) {
	if a > b {
		if a > c {
			mx = a
		} else {
			mx = c
		}
		if b < c {
			mn = b
		} else {
			mn = c
		}
	} else {
		if b > c {
			mx = b
		} else {
			mx = c
		}
		if a < c {
			mn = a
		} else {
			mn = c
		}
	}
	return
}

func calcMinMaxImage(src vision.RGBImage) (mn, mx vision.RGBImage) {
	mn.Resize(src.Size)
	mx.Resize(src.Size)
	
	w1 := src.Width - 1
	
	for y := 0; y < src.Height; y++ {
		start := y * src.Width
		
		for c := 0; c < 3; c++ {
			vl := src.Pixels[start][c]
			vlRight := byte((int(vl) + int(src.Pixels[start + 1][c]))/2)
			mn.Pixels[start][c], mx.Pixels[start][c] = minmax(vl, vl, vlRight)
		}
		
		for x := 1; x < w1; x++ {
			startX := start + x
			for c := 0; c < 3; c++ {
				vl := src.Pixels[startX][c]
				vlLeft := byte((int(vl) + int(src.Pixels[startX - 1][c]))/2)
				vlRight := byte((int(vl) + int(src.Pixels[startX + 1][c]))/2)
				mn.Pixels[startX][c], mx.Pixels[startX][c] = minmax(vlLeft, vl, vlRight)
			}
		}
		startW1 := start + w1
		for c := 0; c < 3; c++ {
			vl := src.Pixels[startW1][c]
			vlLeft := byte((int(vl) + int(src.Pixels[startW1 - 1][c]))/2)
			mn.Pixels[startW1][c], mx.Pixels[startW1][c] = minmax(vlLeft, vl, vl)
		}
	}
	return
}

func calcDSI(left, right vision.RGBImage, maxD int, trimDiff, outDiff byte) (dsi vision.GrayImage) {
	leftMn, leftMx := calcMinMaxImage(left)
	vision.SaveImageAsPng(leftMn.AsImage(), "/tmp/leftmn.png")
	vision.SaveImageAsPng(leftMx.AsImage(), "/tmp/leftmx.png")
	rightMn, rightMx := calcMinMaxImage(right)
	
	l := left.Area()
	
	dsi.Resize(vision.Size{l, maxD + 1})
	for d := 0; d <= maxD; d++ {
		dsiStart := d * l
		for y := 0; y < left.Height; y++ {
			yStart := y * left.Width
			for i := 0; i < d; i++ {
				dsi.Pixels[dsiStart + yStart + i] = outDiff
			}

			for x := d; x < left.Width; x++ {
				xyIdxL := yStart + x
				xyIdxR := xyIdxL - d
				diff := 0
				for c := 0; c < 3; c++ {
					clLeft := left.Pixels[xyIdxL][c]
					mnRight, mxRight := rightMn.Pixels[xyIdxR][c], rightMx.Pixels[xyIdxR][c]
					
					var diffLeft byte
					if clLeft < mnRight {
						diffLeft = mnRight - clLeft
					} else if clLeft > mxRight {
						diffLeft = clLeft - mxRight
					}
					
					clRight := right.Pixels[xyIdxR][c]
					mnLeft, mxLeft := leftMn.Pixels[xyIdxL][c], leftMx.Pixels[xyIdxL][c]
					
					var diffRight byte
					if clRight < mnLeft {
						diffRight = mnLeft - clRight
					} else if clRight > mxLeft {
						diffRight = clRight - mxLeft
					}
					
					var diffC = trimDiff
					if diffLeft < diffC {
						diffC= diffLeft
					}
					if diffRight < diffC {
						diffC = diffRight
					}
					
					diff += int(diffC)
				}
				dsi.Pixels[dsiStart + xyIdxL] = byte(diff / 3)
			}
		}
	}
	return
}

func RGBMatch(left, right vision.RGBImage, opt RGBOption) (disp vision.GrayImage) {
	segms := rgbSegment(left, opt.Segment)
	_ = segms
	dsi := calcDSI(left, right, opt.MaxDisp, 15, 5)
	_ = dsi
	
	log.Println(dsi.Size)
	
	
	return vision.GrayImage{}
}
