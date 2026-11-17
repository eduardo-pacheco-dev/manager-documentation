package httpapi

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"mime/multipart"
)

const formThumbnailMaxEdge = 400

// makeFormThumbnail downscales an uploaded image to a small JPEG used by the
// list, cards and evaluation previews. Full resolution is kept for the
// lightbox/original download.
func makeFormThumbnail(contents []byte) ([]byte, bool) {
	config, _, err := image.DecodeConfig(bytes.NewReader(contents))
	if err != nil || config.Width < 1 || config.Height < 1 {
		return nil, false
	}
	if config.Width*config.Height > 40_000_000 {
		return nil, false
	}
	source, _, err := image.Decode(bytes.NewReader(contents))
	if err != nil {
		return nil, false
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	scale := math.Min(1, float64(formThumbnailMaxEdge)/float64(max(width, height)))
	targetWidth := max(1, int(math.Round(float64(width)*scale)))
	targetHeight := max(1, int(math.Round(float64(height)*scale)))

	target := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.Draw(target, target.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	for y := 0; y < targetHeight; y++ {
		sourceY := bounds.Min.Y + y*height/targetHeight
		for x := 0; x < targetWidth; x++ {
			sourceX := bounds.Min.X + x*width/targetWidth
			target.Set(x, y, source.At(sourceX, sourceY))
		}
	}
	var output bytes.Buffer
	if err := jpeg.Encode(&output, target, &jpeg.Options{Quality: 72}); err != nil {
		return nil, false
	}
	return output.Bytes(), true
}

func (s *Server) storeFormThumbnail(ctx context.Context, header *multipart.FileHeader) string {
	file, err := header.Open()
	if err != nil {
		return ""
	}
	contents, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		return ""
	}
	thumbnail, ok := makeFormThumbnail(contents)
	if !ok {
		return ""
	}
	result, err := s.Uploads.SaveThumbnail(ctx, thumbnail, "uploads/forms/thumbs")
	if err != nil {
		return ""
	}
	return result.URL
}
