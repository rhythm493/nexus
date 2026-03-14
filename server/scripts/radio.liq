# Nexus Radio configuration for Liquidsoap
settings.server.telnet := true
settings.server.telnet.bind_addr := "127.0.0.1"
settings.server.telnet.port := 1234
settings.log.level := 3

# Queue source
queue = request.queue(id="queue")

# Safe fallback when queue is empty
radio = mksafe(queue)

# Output to HTTP harbor with Icy-Metadata
output.harbor(
  mount="/stream",
  port=8080,
  metaint=8192,
  %mp3(bitrate=192, stereo=true, samplerate=44100),
  radio
)

print("Radio server started!")
