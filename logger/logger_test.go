package logger

func TestLogger() {
	log := NewFactorLog()
	log.ERROR("Severity: Error occurred")
	log.WARN("Severity: Warning!!!")
	log.INFO("Severity: I have some info for you")
	log.DEBUG("Severity: Debug what?")
	// log.STACK("Stack from func")
}
