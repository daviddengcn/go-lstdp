package lstdp

import (
	"log"

	"github.com/daviddengcn/go-vision"
)

type SegmentOpt struct {
	T          int
	MaxAdjustX int
	Rx, Ry     int
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
	return sumabs3(int(a[0])-int(b[0]), int(a[1])-int(b[1]), int(a[2])-int(b[2]))
}

func reverse(b byte) byte {
	j := byte(0x80)
	res := byte(0)
	for i := 1; i < 0x100; i <<= 1 {
		if int(b)&i != 0 {
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
		start := y * img.Width

		// initial marks
		marks.Pixels[start] = 255

		mn, mx := img.Pixels[start], img.Pixels[start]
		for x := 1; x < img.Width; x++ {
			p := img.Pixels[start+x]

			toMark := false
			for i := 0; i < 3; i++ {
				if p[i] < mn[i] {
					mn[i] = p[i]
				} else if p[i] > mx[i] {
					mx[i] = p[i]
				}
				if int(mx[i]-mn[i]) > opt.T {
					toMark = true
					break
				}
			}
			if toMark {
				marks.Pixels[start+x] = 255
				mn, mx = p, p
			}
		}

		// repositioning
		for x := 1; x < img.Width; x++ {
			if marks.Pixels[start+x] == 0 {
				continue
			}

			maxDiff := rgbDiff(img.Pixels[start+x], img.Pixels[start+x-1])
			maxX := x
			for x1 := x - 1; x1 >= x-opt.MaxAdjustX && x1 > 0; x1-- {
				if marks.Pixels[start+x1] != 0 {
					break
				}
				diff := rgbDiff(img.Pixels[start+x1], img.Pixels[start+x1-1])
				if diff > maxDiff {
					maxDiff, maxX = diff, x1
				}
			}
			for x1 := x + 1; x1 <= x+opt.MaxAdjustX && x1 < img.Width; x1++ {
				if marks.Pixels[start+x1] != 0 {
					break
				}
				diff := rgbDiff(img.Pixels[start+x1], img.Pixels[start+x1-1])
				if diff > maxDiff {
					maxDiff, maxX = diff, x1
				}
			}

			if maxX != x {
				marks.Pixels[start+x], marks.Pixels[start+maxX] = 0, 255
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
							if marks.Pixels[y1*marks.Width+x1] != 0 {
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
	for i := 0; i < l; i++ {
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
			vlRight := byte((int(vl) + int(src.Pixels[start+1][c])) / 2)
			mn.Pixels[start][c], mx.Pixels[start][c] = minmax(vl, vl, vlRight)
		}

		for x := 1; x < w1; x++ {
			startX := start + x
			for c := 0; c < 3; c++ {
				vl := src.Pixels[startX][c]
				vlLeft := byte((int(vl) + int(src.Pixels[startX-1][c])) / 2)
				vlRight := byte((int(vl) + int(src.Pixels[startX+1][c])) / 2)
				mn.Pixels[startX][c], mx.Pixels[startX][c] = minmax(vlLeft, vl, vlRight)
			}
		}
		startW1 := start + w1
		for c := 0; c < 3; c++ {
			vl := src.Pixels[startW1][c]
			vlLeft := byte((int(vl) + int(src.Pixels[startW1-1][c])) / 2)
			mn.Pixels[startW1][c], mx.Pixels[startW1][c] = minmax(vlLeft, vl, vl)
		}
	}
	return
}

func calcDSI(left, right vision.RGBImage, maxD byte, trimDiff, outDiff byte) (dsi vision.GrayImage) {
	leftMn, leftMx := calcMinMaxImage(left)
	vision.SaveImageAsPng(leftMn.AsImage(), "/tmp/leftmn.png")
	vision.SaveImageAsPng(leftMx.AsImage(), "/tmp/leftmx.png")
	rightMn, rightMx := calcMinMaxImage(right)
	vision.SaveImageAsPng(rightMn.AsImage(), "/tmp/rightmn.png")
	vision.SaveImageAsPng(rightMx.AsImage(), "/tmp/rightmx.png")

	l := left.Area()

	dsi.Resize(vision.Size{l, int(maxD + 1)})
	for d := byte(0); d <= maxD; d++ {
		dsiStart := int(d) * l
		for y := 0; y < left.Height; y++ {
			yStart := y * left.Width
			for i := 0; i < int(d); i++ {
				dsi.Pixels[dsiStart+yStart+i] = outDiff
			}

			for x := int(d); x < left.Width; x++ {
				xyIdxL := yStart + x
				xyIdxR := xyIdxL - int(d)
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
						diffC = diffLeft
					}
					if diffRight < diffC {
						diffC = diffRight
					}

					diff += int(diffC)
				}
				dsi.Pixels[dsiStart+xyIdxL] = byte(diff / 3)
			}
		}
	}
	return
}

type edge struct {
	From int
	To   int
}

type edges struct {
	Edges []edge
	Begin int
}

const (
	DISP_MAX      = 240
	EDGE_DIFF_MAX = 1024
)

type segmPos struct {
	XBeg, XEnd  int
	Y           int
}

type segmNode struct {
	// Position information
	segmPos
	
	// Average color of this segment
	Color       vision.RGB
	
	// Index of the first child
	FirstChild  int
	// Index of the next brother
	NextBrother int

	// Es[d] is the matching energe of current segment with disparity == d.
	Es [DISP_MAX + 1]int

	// E[d] and D[d] store information of the optimal configuration when parent choose disparity to d
	// E[d] is the energe of current subtree and neighboring energe to parent
	E [DISP_MAX + 1]int
	// D[d] is the disparity of the current segment in the optimal configuration.
	D [DISP_MAX + 1]byte
}

func (sn *segmNode) setColorBySum(sum vision.IntRGB) {
	l := sn.XEnd - sn.XBeg
	for c, vl := range sum {
		sn.Color[c] = byte(vl / l)
	}
}

func (sn *segmNode) Len() int {
	return sn.XEnd - sn.XBeg
}

func analyzeSegms(left vision.RGBImage, segmImage vision.IntGrayImage, mst []segmNode) (maxLen255 int) {
	var sum vision.IntRGB
	for y := 0; y < left.Height; y++ {
		start := y * left.Width
		for x := 0; x < left.Width; x++ {
			idx := start + x
			curSegm := segmImage.Pixels[idx]
			if x == 0 || curSegm != segmImage.Pixels[idx - 1] {
				mst[curSegm].XBeg = x
				mst[curSegm].Y = y
				
				if curSegm > 0 {
					mst[curSegm - 1].setColorBySum(sum)
					sum = vision.IntRGB{}
				}
			}
			
			for c := range sum {
				sum[c] += int(left.Pixels[idx][c])
			}
			mst[curSegm].XEnd = x + 1
		}
	}
	
	// Update color of the last segment
	curSegm := segmImage.Pixels[segmImage.Area() - 1]
	mst[curSegm - 1].setColorBySum(sum)
	
	maxLen := 0
	for _, sn := range mst {
		if sn.Len() > maxLen {
			maxLen = sn.Len()
		}
	}
	return maxLen * 255
}

// s1 and s2 are known to be overlapped
func calcSegmOverlap(s1, s2 segmPos) int {
	if s1.Y == s2.Y {
		return 1
	}
	beg := s1.XBeg
	if s2.XBeg > beg {
		beg = s2.XBeg
	}
	
	end := s1.XEnd
	if s2.XEnd < end {
		end = s2.XEnd
	}
	
	return end - beg
}

func insertEdge(edgesList []edges, mst []segmNode, fromSegm, toSegm int, maxLen255 int, minEdge int) (newMinEdge int) {
	if mst[toSegm].FirstChild != 0 {
		return minEdge
	}
	
	l := calcSegmOverlap(mst[fromSegm].segmPos, mst[toSegm].segmPos)
	diff := (maxLen255 - l*(255 - rgbDiff(mst[fromSegm].Color, mst[toSegm].Color) / 3)) * EDGE_DIFF_MAX / maxLen255
	
	edgesList[diff].Edges = append(edgesList[diff].Edges, edge{fromSegm, toSegm})
	if diff < minEdge {
		minEdge = diff
	}
	return minEdge
}

func appendEdges(edgesList []edges, minEdge int, mst []segmNode, segmImage vision.IntGrayImage, curSegm int, maxLen255 int) (newEdgeMin int) {
	x, y := mst[curSegm].XBeg, mst[curSegm].Y
	mst[curSegm].FirstChild = -1
	
	if x > 0 {
		// left
		minEdge = insertEdge(edgesList, mst, curSegm, curSegm - 1, maxLen255, minEdge)
	}
	
	start := y * segmImage.Width
	lastUp, lastDown := -1, -1
	for ;x < mst[curSegm].XEnd; x++ {
		if y > 0 {
			// up
			upSegm := segmImage.Pixels[start - segmImage.Width + x]
			if upSegm != lastUp {
				minEdge = insertEdge(edgesList, mst, curSegm, upSegm, maxLen255, minEdge)
				lastUp = upSegm
			}
		}

		if y < segmImage.Height - 1 {
			// down		
			downSegm := segmImage.Pixels[start + segmImage.Width + x]
			if downSegm != lastDown {
				minEdge = insertEdge(edgesList, mst, curSegm, downSegm, maxLen255, minEdge)
				lastDown = downSegm
			}
		}		
	}
	
	if x < segmImage.Width - 1 {
		// right
		minEdge = insertEdge(edgesList, mst, curSegm, curSegm + 1, maxLen255, minEdge)
	}
	
	return minEdge
}

func findEdge(edgesList []edges, minEdge int, mst []segmNode) int {
	for {
		for edgesList[minEdge].Begin < len(edgesList[minEdge].Edges) {
			eg := &edgesList[minEdge].Edges[edgesList[minEdge].Begin]
			if mst[eg.To].FirstChild == 0 {
				return minEdge
			}
			
			edgesList[minEdge].Begin++
		}
		
		minEdge++
	}
}

func calcMST(left vision.RGBImage, segmImage vision.IntGrayImage) (mst []segmNode) {
	l := segmImage.Area()
	
	nSegm := segmImage.Pixels[l - 1] + 1
	
	log.Println("nSegm", nSegm)
	
	mst = make([]segmNode, nSegm)
	maxLen255 := analyzeSegms(left, segmImage, mst)
	
	log.Println("maxLen", maxLen255/255)
	
	var edgesList [EDGE_DIFF_MAX]edges
	
	// append left-top segment as the root
	minEdge := appendEdges(edgesList[:], EDGE_DIFF_MAX + 1, mst, segmImage, 0, maxLen255)

	for i := 1; i < nSegm; i++ {
		minEdge = findEdge(edgesList[:], minEdge, mst)
		edges := &edgesList[minEdge]
		edge := edges.Edges[edges.Begin]
		edges.Begin++
		
		curSegm := edge.To
		// insert curSegm as first of the children of edge.From.
		mst[curSegm].NextBrother = mst[edge.From].FirstChild
		mst[edge.From].FirstChild = curSegm
		
		minEdge = appendEdges(edgesList[:], minEdge, mst, segmImage, curSegm, maxLen255)
	}	
	
	return mst
}

func aggrDSI(dsi vision.GrayImage, mst []segmNode, width int) {
	for i, node := range mst {
		for d := 0; d < dsi.Height; d++ {
			sum := 0
			start := d*dsi.Width + node.Y*width
			for x := node.XBeg; x < node.XEnd; x++ {
				sum += int(dsi.Pixels[start + x])
			}
			mst[i].Es[d] = sum
		}
	}
}

func dpCalcEv(mst []segmNode, rootSegm, curSegm int, opt Option) {
	var buf [DISP_MAX + 1]int
	for d := byte(0); d <= opt.MaxD; d++ {
		buf[d] = mst[curSegm].Es[d]
	}
	
	for child := mst[curSegm].FirstChild; child != -1; child = mst[child].NextBrother {
		// Recursively update children's energy
		dpCalcEv(mst, curSegm, child, opt)
		
		for d := byte(0); d <= opt.MaxD; d++ {
			buf[d] += mst[child].E[d]
		}
	}
	
	minE, bestD := buf[0], byte(0)
	for d := byte(0); d <= opt.MaxD; d++ {
		if buf[d] < minE {
			minE, bestD = buf[d], d
		}
	}
	
	
	clDiff := rgbDiff(mst[rootSegm].Color, mst[curSegm].Color) / 3
	if clDiff > 128 {
		clDiff = 128
	}
	curT := opt.T1 + opt.T * (128 - clDiff) / 128
	ovLen := calcSegmOverlap(mst[rootSegm].segmPos, mst[curSegm].segmPos)
	
	minE += curT * ovLen
	E1 := opt.T1 * ovLen
	
	for d := byte(0); d <= opt.MaxD; d++ {
		// The case current segment choose its own optimal
		curE, curD := minE, bestD
		
		// The case current segment and parent choose same disparity, smooth energy is supposed to be zero
		if buf[d] < curE {
			curE, curD = buf[d], d
		}
		
		if d > 0 {
			// The case current segment choose one less than parent disparity
			e := buf[d - 1] + E1
			if e < curE {
				curE, curD = e, d - 1
			}
		}
		
		if d < opt.MaxD {
			// The case current segment choose one greater than parent disparity
			e := buf[d + 1] + E1
			if e < curE {
				curE, curD = e, d + 1
			}
		}
		
		mst[curSegm].E[d], mst[curSegm].D[d] = curE, curD
	}
}

func fillDisp(mst []segmNode, disp vision.GrayImage, segm int, rootD byte) {
	curD := mst[segm].D[rootD]
	start := mst[segm].Y * disp.Width
	for x := mst[segm].XBeg; x < mst[segm].XEnd; x++ {
		disp.Pixels[start + x] = curD
	}
	
	for child := mst[segm].FirstChild; child != -1; child = mst[child].NextBrother {
		fillDisp(mst, disp, child, curD)
	}
}

func dpRoot(mst []segmNode, disp vision.GrayImage, opt Option) {
	var buf [DISP_MAX + 1]int
	for d := byte(0); d <= opt.MaxD; d++ {
		buf[d] = mst[0].Es[d]
	}
	
	for child := mst[0].FirstChild; child != -1; child = mst[child].NextBrother {
		dpCalcEv(mst, 0, child, opt)
		
		for d := byte(0); d <= opt.MaxD; d++ {
			buf[d] += mst[child].E[d]
		}
	}
	
	minE, bestD := buf[0], byte(0)
	for d := byte(0); d <= opt.MaxD; d++ {
		if buf[d] < minE {
			minE, bestD = buf[d], d
		}
	}
	mst[0].D[bestD] = bestD
	
	fillDisp(mst, disp, 0, bestD)
}

func RGBMatch(left, right vision.RGBImage, opt RGBOption) (disp vision.GrayImage) {
	segms := rgbSegment(left, opt.Segment)
	mst := calcMST(left, segms)
	
	dsi := calcDSI(left, right, opt.MaxD, 15, 3)
	
	aggrDSI(dsi, mst, left.Width)
	
	log.Println(dsi.Size)
	
	/*
	 * Tree-DP
	 */
	disp.Resize(left.Size)
	dpRoot(mst, disp, opt.Option)

	var mdisp vision.GrayImage
	mdisp.Resize(disp.Size)
	for i, d := range disp.Pixels {
		mdisp.Pixels[i] = byte(int(d)*255 / int(opt.MaxD))
	}
	err := vision.SaveImageAsPng(mdisp.AsImage(), "/tmp/disp.png")
	log.Printf("%v", err)
	
	return disp
}
