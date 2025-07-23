# Troubleshooting Guide

## Double Puppeting Issues

### Host Messages Appear as Received (Not Sent)

**Symptoms:**
- Messages you send from Hostex appear as received messages from the bridge bot
- You see your own messages as coming from `@sh-hostexbot:beeper.local`

**Solution:**
1. **Check your configuration**: Ensure you're using the bridgev2 config format
2. **Verify double puppeting setup**:
   ```yaml
   double_puppet:
     secrets:
       beeper.com: "as_token:your_actual_as_token_here"
   ```
3. **Check bridge registration**: Ensure `registration.yaml` matches your config tokens
4. **Restart bridge**: `kill <bridge_pid>` and restart with `./mautrix-hostex -c config.yaml`

### Bridge Bot Username Issues

**Symptoms:**
- Bridge fails to start with registration errors
- Error: "The as_token was accepted, but the /register request was not"

**Solution:**
1. **Check bot username**: Should be `sh-hostexbot` in config.yaml
2. **Verify registration**: Check `registration.yaml` has matching `sender_localpart: sh-hostexbot`
3. **Check username template**: Should be `sh-hostex_{{.}}` in config.yaml

### Configuration Format Issues

**Symptoms:**
- Bridge starts but double puppeting doesn't work
- Missing network configuration

**Solution:**
1. **Use bbctl**: Generate config with `bbctl config --type bridgev2 sh-generic -o config.yaml`
2. **Add network section**:
   ```yaml
   network:
     hostex_api_url: https://api.hostex.io/v3
     admin_user: "@yourusername:beeper.com"
   ```
3. **Update bridge naming**: Ensure all references use `sh-hostex` prefix

## API Issues

### Subscription Expired Error

**Symptoms:**
- Error: "API error 420: Oops! Your subscription has expired"
- Bridge connects but can't fetch conversations

**Solution:**
1. Log into your Hostex account
2. Renew your API subscription
3. Restart the bridge - no configuration changes needed

### Authentication Failures

**Symptoms:**
- Error: "Failed to authenticate with Hostex API"
- HTTP 401 errors in logs

**Solution:**
1. **Check API token**: Verify your Hostex API token is still valid
2. **Token format**: Ensure token doesn't have extra whitespace or quotes
3. **API URL**: Verify using `https://api.hostex.io/v3` in network config

## Bridge Connectivity Issues

### Websocket Connection Failures

**Symptoms:**
- Bridge fails to connect to Matrix homeserver
- "Failed to connect to homeserver" errors

**Solution:**
1. **Check homeserver address**: Should be `https://matrix.beeper.com/_hungryserv/yourusername`
2. **Verify tokens**: Ensure `as_token` and `hs_token` match registration.yaml
3. **Network connectivity**: Test connection to Beeper servers

### Room Creation Issues

**Symptoms:**
- Conversations exist in Hostex but no Matrix rooms created
- Bridge logs show "Portal not found" repeatedly

**Solution:**
1. **Check permissions**: Ensure bridge has permission to create rooms
2. **Force sync**: Use `!hostex refresh` command in bridge management room
3. **Check logs**: Look for specific error messages about room creation

### Command Processing Issues

**Symptoms:**
- Bridge commands like `!hostex refresh` or `help` don't work
- Commands are ignored or show no response
- Logs show "unencrypted message" errors with "FAIL_RETRIABLE" status

**Solution:**
1. **Check encryption settings**: In config.yaml, ensure `encryption.require: false`
   ```yaml
   encryption:
     allow: true
     default: true
     require: false  # Must be false to process unencrypted commands
   ```
2. **Restart bridge**: Kill and restart the bridge process after config changes
3. **Test commands**: Try `help` command first to verify basic functionality

**Background:**
The bridge may be configured to require all messages to be encrypted, but commands are often sent unencrypted. Setting `require: false` allows the bridge to process both encrypted room messages and unencrypted commands.

## Debugging Steps

### Enable Debug Logging

1. **Set log level**: In config.yaml, set `min_level: debug`
2. **Check logs**: Monitor `logs/bridge.log` for detailed information
3. **Look for patterns**: Focus on errors related to double puppeting or API calls

### Verify Configuration

```bash
# Check your configuration matches expected format
grep -A 5 "double_puppet:" config.yaml
grep -A 5 "network:" config.yaml
grep "username.*hostex" config.yaml
```

### Test API Connection

```bash
# Test your Hostex API token
curl -H "Authorization: Bearer your_token_here" https://api.hostex.io/v3/properties
```

### Bridge Status Check

```bash
# Check if bridge is running
ps aux | grep mautrix-hostex

# Check websocket connection in logs
tail -f logs/bridge.log | grep -i websocket
```

## Getting Help

If issues persist after trying these solutions:

1. **Check logs**: Include relevant log snippets when asking for help
2. **Configuration**: Verify your configuration follows the examples
3. **Version**: Ensure you're using the latest version with double puppeting fixes
4. **GitHub Issues**: Report bugs at https://github.com/keithah/matrix-hostex-bridge/issues

## Common Log Patterns

### Successful Double Puppeting
```
Successfully shared keys
Setting contact info on the appservice bot
Starting Hostex connector
Custom command handlers ENABLED for room cleanup
```

### Failed Double Puppeting
```
The as_token was accepted, but the /register request was not
Failed to authenticate with Hostex API
```

### API Connection Success
```
Checking conversations for new messages
Processing existing Matrix room for new messages
Successfully sent message to Hostex
```