package agorasdk

func validateTranscodeVideo(frame EncodedVideoFrame) error {
	if err := frame.validate(); err != nil {
		return err
	}
	if frame.Codec != H264 && frame.Codec != H265 {
		return invalid("transcode codec", "requires H264 or H265")
	}
	return nil
}
