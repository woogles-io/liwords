package mod

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/emailer"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/notify"
	"github.com/domino14/liwords/pkg/user"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/rs/zerolog/log"
)

func sendNotification(ctx context.Context, us user.Store, user *entity.User, action *ms.ModAction, notorietyReport string) {
	actionEmailText, ok := ModActionEmailMap[action.Type]
	if !ok {
		return
	}
	config, ok := ctx.Value(config.CtxKeyword).(*config.Config)
	if !ok {
		log.Err(errors.New("config does not exist in notify")).Str("userID", user.UUID).Msg("nil-config")
		return
	}
	if config.MailgunKey != "" && !IsRemoval(action) {
		emailContent, emailSubject, err := instantiateEmail(user.Username,
			actionEmailText,
			action.Note,
			action.StartTime,
			action.EndTime,
			action.EmailType)
		if err == nil {
			go func() {
				_, err := emailer.SendSimpleMessage(config.MailgunKey,
					user.Email,
					emailSubject,
					emailContent)
				if err != nil {
					// Errors should not be fatal, just log them
					log.Err(err).Str("userID", user.UUID).Msg("mod-action-send-user-email")
				}
			}()
		} else {
			log.Err(err).Str("userID", user.UUID).Msg("mod-action-generate-user-email")
		}

	}
	if config.DiscordToken != "" {
		var modUsername string
		var err error
		if action.ApplierUserId == AutomodUserId {
			modUsername = AutomodUserId
		} else {
			modUser, err := us.GetByUUID(ctx, action.ApplierUserId)
			if err != nil {
				log.Err(err).Str("userID", user.UUID).Msg("mod-action-applier")
				return
			}
			modUsername = modUser.Username
		}

		var message string
		if action.ApplierUserId == action.UserId && action.Type == ms.ModActionType_SUSPEND_ACCOUNT {
			message = fmt.Sprintf("User %s has deleted their account.", user.Username)
		} else {
			appliedOrRemoved := "applied to"
			actionExpiration := "action will expire"
			if IsRemoval(action) {
				appliedOrRemoved = "removed from"
				actionExpiration = "will take effect"
			}
			message = fmt.Sprintf("Action %s was %s user %s by moderator %s.", actionEmailText, appliedOrRemoved, user.Username, modUsername)
			_, actionExists := ModActionDispatching[action.Type]
			if !actionExists { // Action is non-transient
				if action.Duration == 0 {
					message += " This action is permanent."
				} else if action.EndTime == nil {
					log.Err(err).Str("userID", user.UUID).Msg("mod-action-endtime-nil")
				} else {
					golangActionEndTime := action.EndTime.AsTime()
					message += fmt.Sprintf(" This %s on %s.", actionExpiration, golangActionEndTime.UTC().Format(time.UnixDate))
				}
			}
		}
		notify.Post(message+notorietyReport, config.DiscordToken)
	}
}
