package quota

import (
	"os"
	"strconv"
	"strings"

	userdomain "github.com/ajuda-dev/backend/src/service/identity/domain"
)

const (
	defaultMaxOwnedCommunities           = 5
	defaultMaxOwnedCommunitiesMod        = 10
	defaultMaxPendingEvents              = 5
	defaultMaxPendingEventsMod           = 10
	defaultMaxActiveEvents               = 20
	defaultMaxActiveEventsMod            = 40
	defaultMaxCommunityMemberships       = 100
	defaultMaxCommunityMembershipsMod    = 200
	defaultRateEventCreatePerHour        = 10
	defaultRateEventCreatePerHourMod     = 20
	defaultRateCommunityCreatePerHour    = 3
	defaultRateCommunityCreatePerHourMod = 6
	defaultRateCommunityJoinPerHour      = 30
	defaultRateCommunityJoinPerHourMod   = 60
)

const (
	BucketEventCreate     = "event_create"
	BucketCommunityCreate = "community_create"
	BucketCommunityJoin   = "community_join"
)

type Config struct {
	MaxOwnedCommunities                 int
	MaxOwnedCommunitiesModerator        int
	MaxPendingEvents                    int
	MaxPendingEventsModerator           int
	MaxActiveEvents                     int
	MaxActiveEventsModerator            int
	MaxCommunityMemberships             int
	MaxCommunityMembershipsModerator    int
	RateEventCreatePerHour              int
	RateEventCreatePerHourModerator     int
	RateCommunityCreatePerHour          int
	RateCommunityCreatePerHourModerator int
	RateCommunityJoinPerHour            int
	RateCommunityJoinPerHourModerator   int
	MaxSkillsPerUser                    *int
	MaxSkillsPerUserModerator           *int
}

func LoadFromEnv() Config {
	return Config{
		MaxOwnedCommunities:                 intFromEnv("MAX_OWNED_COMMUNITIES", defaultMaxOwnedCommunities),
		MaxOwnedCommunitiesModerator:        intFromEnv("MAX_OWNED_COMMUNITIES_MODERATOR", defaultMaxOwnedCommunitiesMod),
		MaxPendingEvents:                    intFromEnv("MAX_PENDING_EVENTS", defaultMaxPendingEvents),
		MaxPendingEventsModerator:           intFromEnv("MAX_PENDING_EVENTS_MODERATOR", defaultMaxPendingEventsMod),
		MaxActiveEvents:                     intFromEnv("MAX_ACTIVE_EVENTS", defaultMaxActiveEvents),
		MaxActiveEventsModerator:            intFromEnv("MAX_ACTIVE_EVENTS_MODERATOR", defaultMaxActiveEventsMod),
		MaxCommunityMemberships:             intFromEnv("MAX_COMMUNITY_MEMBERSHIPS", defaultMaxCommunityMemberships),
		MaxCommunityMembershipsModerator:    intFromEnv("MAX_COMMUNITY_MEMBERSHIPS_MODERATOR", defaultMaxCommunityMembershipsMod),
		RateEventCreatePerHour:              intFromEnv("RATE_LIMIT_EVENT_CREATE_PER_HOUR", defaultRateEventCreatePerHour),
		RateEventCreatePerHourModerator:     intFromEnv("RATE_LIMIT_EVENT_CREATE_PER_HOUR_MODERATOR", defaultRateEventCreatePerHourMod),
		RateCommunityCreatePerHour:          intFromEnv("RATE_LIMIT_COMMUNITY_CREATE_PER_HOUR", defaultRateCommunityCreatePerHour),
		RateCommunityCreatePerHourModerator: intFromEnv("RATE_LIMIT_COMMUNITY_CREATE_PER_HOUR_MODERATOR", defaultRateCommunityCreatePerHourMod),
		RateCommunityJoinPerHour:            intFromEnv("RATE_LIMIT_COMMUNITY_JOIN_PER_HOUR", defaultRateCommunityJoinPerHour),
		RateCommunityJoinPerHourModerator:   intFromEnv("RATE_LIMIT_COMMUNITY_JOIN_PER_HOUR_MODERATOR", defaultRateCommunityJoinPerHourMod),
		MaxSkillsPerUser:                    optionalIntFromEnv("MAX_SKILLS_PER_USER"),
		MaxSkillsPerUserModerator:           optionalIntFromEnv("MAX_SKILLS_PER_USER_MODERATOR"),
	}
}

func LimitForRole(role string, userLimit, moderatorLimit int) (limit int, bypass bool) {
	if role == userdomain.UserRoleAdmin {
		return 0, true
	}
	if role == userdomain.UserRoleModerator {
		return moderatorLimit, false
	}
	return userLimit, false
}

func OptionalLimitForRole(role string, userLimit, moderatorLimit *int) (limit int, enforce bool, bypass bool) {
	if role == userdomain.UserRoleAdmin {
		return 0, false, true
	}
	var selected *int
	if role == userdomain.UserRoleModerator {
		selected = moderatorLimit
	} else {
		selected = userLimit
	}
	if selected == nil {
		return 0, false, false
	}
	return *selected, true, false
}

func intFromEnv(key string, defaultValue int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultValue
	}
	return n
}

func optionalIntFromEnv(key string) *int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}
