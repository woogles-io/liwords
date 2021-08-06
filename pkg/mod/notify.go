package mod

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/emailer"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog/log"
)

func notify(ctx context.Context, us user.Store, user *entity.User, action *ms.ModAction, notorietyReport string) {
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
		emailContent, err := instantiateEmail(user.Username,
			actionEmailText,
			action.Note,
			action.StartTime,
			action.EndTime,
			action.EmailType)
		if err == nil {
			_, err := emailer.SendSimpleMessage(config.MailgunKey,
				user.Email,
				fmt.Sprintf("Woogles Terms of Service Violation for Account %s", user.Username),
				emailContent)
			if err != nil {
				// Errors should not be fatal, just log them
				log.Err(err).Str("userID", user.UUID).Msg("mod-action-send-user-email")
			}
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

		appliedOrRemoved := "applied to"
		actionExpiration := "action will expire"
		if IsRemoval(action) {
			appliedOrRemoved = "removed from"
			actionExpiration = "will take effect"
		}
		message := fmt.Sprintf("Action %s was %s user %s by moderator %s.", actionEmailText, appliedOrRemoved, user.Username, modUsername)
		_, actionExists := ModActionDispatching[action.Type]
		if !actionExists { // Action is non-transient
			if action.Duration == 0 {
				message += " This action is permanent."
			} else if action.EndTime == nil {
				log.Err(err).Str("userID", user.UUID).Msg("mod-action-endtime-nil")
			} else {
				golangActionEndTime, err := ptypes.Timestamp(action.EndTime)
				if err != nil {
					log.Err(err).Str("error", err.Error()).Msg("mod-action-endtime-conversion")
				} else {
					message += fmt.Sprintf(" This %s on %s.", actionExpiration, golangActionEndTime.UTC().Format(time.UnixDate))
				}
			}
		}
		requestBody, err := json.Marshal(map[string]string{"content": message + notorietyReport})
		// Errors should not be fatal, just log them
		if err != nil {
			log.Err(err).Str("error", err.Error()).Msg("mod-action-discord-notification-marshal")
		} else {
			resp, err := http.Post(config.DiscordToken, "application/json", bytes.NewBuffer(requestBody))
			// Errors should not be fatal, just log them
			if err != nil {
				log.Err(err).Str("error", err.Error()).Msg("mod-action-discord-notification-post-error")
			} else if resp.StatusCode != 204 { // No Content
				// We do not expect any other response
				log.Err(err).Str("status", resp.Status).Msg("mod-action-discord-notification-post-bad-response")
			}
		}
	}

}
