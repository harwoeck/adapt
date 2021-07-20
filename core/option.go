package core

import (
	logger "github.com/harwoeck/liblog/contract"
)

// Option can modify the behaviour of Migrate and/or provide additional configuration
// values, like custom contract.Logger
type Option func(*exec) error

// DisableHashIntegrityChecks disables the hash integrity checks of SqlStatementsSource
// migrations against the already applied ones. By default adapt always performs these
// checks to protect against unwanted changes to SQL-Statements scripts after they have
// already been applied to your Driver. Disabling it should be done with caution!
func DisableHashIntegrityChecks(r *exec) error {
	r.optDisableHashIntegrityChecks = true
	return nil
}

// DisableDriverLocks disables mutex acquiring/releasing of a Driver, even if the Driver
// itself reports to support locking.
func DisableDriverLocks(r *exec) error {
	r.optDisableDriverLocks = true
	return nil
}

// CustomLogger provides a custom contract.Logger implementation to adapt. It will be
// used within the whole module and passed down to Driver and Source children.
func CustomLogger(log logger.Logger) Option {
	return func(r *exec) error {
		r.log = log
		return nil
	}
}

// DisableLogger fully disables logging output
func DisableLogger(r *exec) error {
	r.log = logger.MustNewStd(logger.DisableLogWrites())
	return nil
}
