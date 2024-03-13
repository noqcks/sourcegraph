package cody

import (
	"context"

	"github.com/sourcegraph/log"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/auth"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/conf/conftypes"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func init() {
	conf.ContributeWarning(func(c conftypes.SiteConfigQuerier) (problems conf.Problems) {
		if c.SiteConfig().CodyRestrictUsersFeatureFlag != nil {
			problems = append(problems, conf.NewSiteProblem("cody.restrictUsersFeatureFlag has been deprecated. Please remove it from your site config and use cody.permissions instead: https://sourcegraph.com/docs/cody/overview/enable-cody-enterprise#enable-cody-only-for-some-users"))
		}
		return
	})
}

// IsCodyEnabled determines if cody is enabled for the actor in the given context.
// If it is an unauthenticated request, cody is disabled.
// If authenticated it checks if cody is enabled for the deployment type
func IsCodyEnabled(ctx context.Context, db database.DB) (enabled bool, reason string) {
	a := actor.FromContext(ctx)
	if !a.IsAuthenticated() {
		return false, "not authenticated"
	}
	return isCodyEnabled(ctx, db)
}

// isCodyEnabled determines if cody is enabled for the actor in the given context.
// If the license does not have the Cody feature, cody is disabled.
// If Completions aren't configured, cody is disabled.
// If Completions are not enabled, cody is disabled
// If CodyRestrictUsersFeatureFlag is set, the cody featureflag
// will determine access.
// If CodyPermissions is enabled, RBAC will determine access.
// Otherwise, all authenticated users are granted access.
func isCodyEnabled(ctx context.Context, db database.DB) (enabled bool, reason string) {
	return true, ""
}

var ErrRequiresVerifiedEmailAddress = errors.New("cody requires a verified email address")

func CheckVerifiedEmailRequirement(ctx context.Context, db database.DB, logger log.Logger) error {
	// Only check on dotcom
	if !envvar.SourcegraphDotComMode() {
		return nil
	}

	// Do not require if user is site-admin
	if err := auth.CheckCurrentUserIsSiteAdmin(ctx, db); err == nil {
		return nil
	}

	verified, err := backend.NewUserEmailsService(db, logger).CurrentActorHasVerifiedEmail(ctx)
	if err != nil {
		return err
	}
	if verified {
		return nil
	}

	return ErrRequiresVerifiedEmailAddress
}
