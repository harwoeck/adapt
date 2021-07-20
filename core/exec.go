package core

import (
	"os"

	logger "github.com/harwoeck/liblog/contract"
)

type exec struct {
	executor string
	driver   Driver
	sources  SourceCollection
	log      logger.Logger

	optDisableDriverLocks         bool
	optDisableHashIntegrityChecks bool
	optAlwaysExecAllMigrations    bool

	driverIsDatabaseDriver                bool
	driverAsDatabaseDriver                DatabaseDriver
	driverIsDatabaseDriverCustomMigration bool
	driverAsDatabaseDriverCustomMigration DatabaseDriverCustomMigration

	available          []*AvailableMigration
	driverLockAcquired bool
	applied            []*Migration
	unknownApplied     []*Migration
}

func newExec(executor string, driver Driver, sources SourceCollection, options ...Option) (*exec, error) {
	// create
	e := &exec{
		executor: executor,
		driver:   driver,
		sources:  sources,
		log: logger.MustNewStd(
			logger.OutWriter(os.Stdout),
			logger.ErrWriter(os.Stdout),
			logger.IsInDevEnvironment(false),
		),
	}

	// apply options
	for _, opt := range options {
		if err := opt(e); err != nil {
			return nil, err
		}
	}

	// name logger
	e.log = e.log.Named("adapt")

	// check if driver is a DatabaseDriver
	if asDB, ok := driver.(DatabaseDriver); ok {
		e.driverIsDatabaseDriver = ok
		e.driverAsDatabaseDriver = asDB

		if asCustomDB, ok := driver.(DatabaseDriverCustomMigration); ok {
			e.driverIsDatabaseDriverCustomMigration = true
			e.driverAsDatabaseDriverCustomMigration = asCustomDB
		}
	}

	return e, nil
}

func (e *exec) run() (err error) {
	defer func() {
		closeErr := e.stageClose()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	err = e.stageInit()
	if err != nil {
		return err
	}

	err = e.stageHealthCheck()
	if err != nil {
		return err
	}

	err = e.stagePrepareLocal()
	if err != nil {
		return err
	}

	err = e.acquireDriverLock()
	if err != nil {
		return err
	}
	if e.driverLockAcquired {
		defer func() {
			unlockErr := e.releaseDriverLock()
			if unlockErr != nil && err == nil {
				err = unlockErr
			}
		}()
	}

	err = e.stagePrepareRemote()
	if err != nil {
		return err
	}

	err = e.stageStart()
	if err != nil {
		return err
	}

	return nil
}
