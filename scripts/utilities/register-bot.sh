curl -X POST -H 'Content-Type: application/json' http://liwords.localhost/twirp/user_service.RegistrationService/Register -d "{\"username\": \"$1\", \"password\": \"password123\", \"email\": \"$1-bot@example.com\"}"

docker-compose exec db psql -U postgres liwords -c "UPDATE users SET internal_bot='t' WHERE username = '$1';"