Types of ZSET

channelpresence:channelname ZSET for every channel. A ZSET is essentially a sorted set. The keys of the set are going to be the player presences: userid#username#connid and the values of the set are the last updated timestamp. Every couple minutes we can run a job that’ll clear out old timestamps for all channels. (easy to do this with ZREMBYRANGE or something). This clears out timestamps when the socket server goes down or crashes. We also clear them out whenever a user leaves a channel. I think this works and avoids using the KEYS command. To get all presences in a channel it’s just ZRANGEBYSCORE channelpresence:channelname -inf +inf .

- this ZSET has an expiry of ~1 minute
- every time we get a ping, we extend the expiry
- we also call ZREMBYRANGE every Xth ping (maybe this could be a probabilistic function) or we can do it every ping as well

<!--
channelpresences is a single ZSET with the channel as the key and the timestamp as the value. Same as userpresences, this contains the master list of all active channels. -->

userpresence:userid is a ZSET that contains presence info for each user. Keys are connid#channel, values are timestamp. To get all channels for a user it’s ZRANGEBYSCORE userpresence:userid -inf +inf
-- this ZSET has a TTL that gets updated whenever we ping

- same as the other zset, we also call ZREMBYRANGE every Xth ping.

userpresences is a single ZSET with the userid as the key and timestamp as the value. This lets us find all users that are online. We can call ZCARD to get count of all users. We can call ZREMBYRANGE to expire old users.

The front end will ping the backend with the userID and connID. The backend will then update the TTL of the key userpresence:userid, scan the set for all elements starting with connid#, and update their timestamps, and then update the timestamps for all these channels (in channelpresence:channelname) that user is in (i.e. the key named userid#connid in each relevant ZSET will be updated with the current timestamp). It also updates the userpresences and channelpresences keys.

AddPresence: same as ping function above essentially.

ClearPresence: go into userpresence:userid and delete all starting with connid#

- then go into these channels in channelpresence:channelname and delete userid#username#connid for each channel

---

<!--
presences, ZSET all with timestamps.

- ucc:userid:connid:channel
- u:userid
- uc:userid:connid
- c:channel

userpresences, ZSET

- u:userid -->

---

requirements:

- channel presence should tell us which users are in that channel
- user presence should tell us what channels this user is in across their different connections
- a hard crash of the socket server (or a deploy) should invalidate/expire these presences eventually.
