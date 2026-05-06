package observability

import "github.com/rs/zerolog"

// CloseOrLog calls c.Close() and logs any non-nil result as a warning.
// Defer it from a composition root so resource cleanup errors during
// shutdown stay observable instead of disappearing into a `defer x.Close()`
// idiom that errcheck (rightly) flags.
//
//	defer observability.CloseOrLog(logger, "redis", rdb)
func CloseOrLog(log zerolog.Logger, name string, c interface{ Close() error }) {
	if err := c.Close(); err != nil {
		log.Warn().Err(err).Str("resource", name).Msg("close failed")
	}
}
