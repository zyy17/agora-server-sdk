package agorartm

// Reason returns the linked SDK's description of this signed error code.
// Unknown codes are delegated to the SDK without normalization or sign changes.
func (code RtmErrorCode) Reason() string { return GetErrorReason(int(code)) }
