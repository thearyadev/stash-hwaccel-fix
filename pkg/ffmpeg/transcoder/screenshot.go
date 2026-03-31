package transcoder

import (
	"os"

	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/logger"
)

type ScreenshotOptions struct {
	OutputPath string
	OutputType ScreenshotOutputType

	// Quality is the quality scale. See https://ffmpeg.org/ffmpeg.html#Main-options
	Quality int

	// Width is the width to scale the screenshot to. If 0, no scaling will be applied.
	Width int
	// Height is the height to scale the screenshot to. If 0, no scaling will be applied.
	// Not used if Width is set.
	Height int

	// Verbosity is the logging verbosity. Defaults to LogLevelError if not set.
	Verbosity ffmpeg.LogLevel

	UseSelectFilter bool

	// VideoCodec is the codec of the input video. Used to determine if hardware decode should be used.
	VideoCodec string
}

func (o *ScreenshotOptions) setDefaults() {
	if o.Verbosity == "" {
		o.Verbosity = ffmpeg.LogLevelError
	}
}

type ScreenshotOutputType struct {
	codec  *ffmpeg.VideoCodec
	format ffmpeg.Format
}

func (t ScreenshotOutputType) Args() []string {
	var ret []string
	if t.codec != nil {
		ret = append(ret, t.codec.Args()...)
	}
	if t.format != "" {
		ret = append(ret, t.format.Args()...)
	}

	return ret
}

var (
	ScreenshotOutputTypeImage2 = ScreenshotOutputType{
		format: "image2",
	}
	ScreenshotOutputTypeBMP = ScreenshotOutputType{
		codec:  &ffmpeg.VideoCodecBMP,
		format: "rawvideo",
	}
)

func ScreenshotTime(input string, t float64, options ScreenshotOptions) ffmpeg.Args {
	options.setDefaults()

	var args ffmpeg.Args
	args = args.LogLevel(options.Verbosity)
	args = args.Overwrite()

	// Add hardware decode args before input if applicable
	args = addHardwareDecodeArgs(args, options.VideoCodec)

	args = args.Seek(t)

	args = args.Input(input)
	args = args.VideoFrames(1)

	if options.Quality > 0 {
		args = args.FixedQualityScaleVideo(options.Quality)
	}

	var vf ffmpeg.VideoFilter

	if options.Width > 0 {
		vf = vf.ScaleWidth(options.Width)
		args = args.VideoFilter(vf)
	} else if options.Height > 0 {
		vf = vf.ScaleHeight(options.Height)
		args = args.VideoFilter(vf)
	}

	args = args.AppendArgs(options.OutputType)
	args = args.Output(options.OutputPath)

	return args
}

// addHardwareDecodeArgs adds hardware decode arguments for AV1 if configured.
// Follows the same pattern as stream_segmented.go for consistency.
func addHardwareDecodeArgs(args ffmpeg.Args, videoCodec string) ffmpeg.Args {
	decodeMethod, decodeMethodExists := os.LookupEnv("FORCE_AV1_HW_DECODE_METHOD")
	if !decodeMethodExists {
		logger.Debug("FORCE_AV1_HW_DECODE_METHOD was not provided. Defaulting to automatic selection")
	}

	if videoCodec == "av1" && decodeMethodExists {
		args = append(args, "-c:v", decodeMethod)
	}

	return args
}

// ScreenshotFrame uses the select filter to get a single frame from the video.
// It is very slow and should only be used for files with very small duration in secs / frame count.
func ScreenshotFrame(input string, frame int, options ScreenshotOptions) ffmpeg.Args {
	options.setDefaults()

	var args ffmpeg.Args
	args = args.LogLevel(options.Verbosity)
	args = args.Overwrite()

	// Add hardware decode args before input if applicable
	args = addHardwareDecodeArgs(args, options.VideoCodec)

	args = args.Input(input)
	args = args.VideoFrames(1)

	args = args.VSync(ffmpeg.VSyncMethodPassthrough)

	var vf ffmpeg.VideoFilter
	// keep only frame number options.Frame)
	vf = vf.Select(frame)

	if options.Width > 0 {
		vf = vf.ScaleWidth(options.Width)
	}

	args = args.VideoFilter(vf)

	args = args.AppendArgs(options.OutputType)
	args = args.Output(options.OutputPath)

	return args
}
