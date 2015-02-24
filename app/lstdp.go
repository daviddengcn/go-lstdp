package main

import (
	"log"
	
	"github.com/daviddengcn/go-lstdp"
	"github.com/daviddengcn/go-vision"
)

func main() {
	fnLeft := "/Users/david/Program/go/src/github.com/daviddengcn/go-lstdp/app/images/venus-im2.png"
	fnRight := "/Users/david/Program/go/src/github.com/daviddengcn/go-lstdp/app/images/venus-im6.png"
	var l, r vision.RGBImage
	if err := l.LoadFromFile(fnLeft); err != nil {
		log.Fatalf("Load file %s failed: ", fnLeft, err)
	}
	if err := r.LoadFromFile(fnRight); err != nil {
		log.Fatalf("Load file %s failed: ", fnRight, err)
	}
	
	var opt lstdp.RGBOption
	opt.Segment.T = 20
	opt.Segment.MaxAdjustX = 2
	opt.Segment.Rx, opt.Segment.Ry = 2, 2
	opt.MaxD = 20
	opt.T = 10
	opt.T1 = 2
	lstdp.RGBMatch(l, r, opt)
}