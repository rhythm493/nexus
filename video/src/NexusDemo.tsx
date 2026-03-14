import React from "react";
import {
  useCurrentFrame,
  useVideoConfig,
  interpolate,
  spring,
  Easing,
} from "remotion";

// ─── Timeline (frames at 30fps) ─────────────────────────
// 0-30:    Title card "Nexus"
// 30-60:   Phone frame slides up
// 60-90:   "Connected to server" appears
// 90-130:  User types "play shape of you"
// 130-170: Thinking bubble appears
// 170-230: Tool call + result animate in
// 230-290: Assistant response appears
// 290-350: "Now playing" card with album art
// 350-420: Second command "set volume to 30"
// 420-480: Tool calls for volume
// 480-530: Sonos visualization
// 530-600: Feature badges fly in
// 600-700: "github.com/rhythm493/nexus" CTA
// 700-750: Fade out

const COLORS = {
  bg: "#0a0a0a",
  card: "#1a1a2e",
  primary: "#6c63ff",
  secondary: "#00d2ff",
  text: "#ffffff",
  textMuted: "#888899",
  userBubble: "#6c63ff",
  assistantBubble: "#1e1e3a",
  thinkingBubble: "#16162a",
  toolBubble: "#0d1b2a",
  success: "#00e676",
  border: "#2a2a4a",
};

// ─── Helper Components ───────────────────────────────────

const FadeIn: React.FC<{
  children: React.ReactNode;
  startFrame: number;
  duration?: number;
  style?: React.CSSProperties;
}> = ({ children, startFrame, duration = 15, style }) => {
  const frame = useCurrentFrame();
  const opacity = interpolate(frame, [startFrame, startFrame + duration], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const y = interpolate(frame, [startFrame, startFrame + duration], [20, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.cubic),
  });
  return (
    <div style={{ opacity, transform: `translateY(${y}px)`, ...style }}>
      {children}
    </div>
  );
};

const TypeWriter: React.FC<{
  text: string;
  startFrame: number;
  speed?: number;
  style?: React.CSSProperties;
}> = ({ text, startFrame, speed = 1.5, style }) => {
  const frame = useCurrentFrame();
  const charsToShow = Math.min(
    text.length,
    Math.max(0, Math.floor((frame - startFrame) * speed))
  );
  return <span style={style}>{text.slice(0, charsToShow)}</span>;
};

// ─── Phone Mockup ────────────────────────────────────────

const PhoneFrame: React.FC<{
  children: React.ReactNode;
  startFrame: number;
  layout?: "vertical" | "horizontal";
}> = ({ children, startFrame, layout = "vertical" }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const slideUp = spring({ frame: frame - startFrame, fps, config: { damping: 15 } });
  const y = interpolate(slideUp, [0, 1], [400, 0]);
  const isH = layout === "horizontal";

  return (
    <div
      style={{
        position: "absolute",
        ...(isH
          ? { left: 140, top: "50%", transform: `translateY(-50%) translateY(${y}px) scale(0.85)` }
          : { top: 180, left: "50%", transform: `translateX(-50%) translateY(${y}px)` }),
        width: 420,
        height: 860,
        borderRadius: 44,
        border: `3px solid ${COLORS.border}`,
        background: COLORS.card,
        overflow: "hidden",
        boxShadow: "0 30px 80px rgba(108, 99, 255, 0.15)",
      }}
    >
      {/* Status bar */}
      <div
        style={{
          height: 44,
          background: "#111128",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          borderBottom: `1px solid ${COLORS.border}`,
        }}
      >
        <span style={{ color: COLORS.textMuted, fontSize: 13, fontFamily: "monospace" }}>
          9:41
        </span>
      </div>

      {/* App bar */}
      <div
        style={{
          height: 56,
          background: "#111128",
          display: "flex",
          alignItems: "center",
          paddingLeft: 20,
          gap: 12,
          borderBottom: `1px solid ${COLORS.border}`,
        }}
      >
        <div
          style={{
            width: 10,
            height: 10,
            borderRadius: "50%",
            background: COLORS.success,
          }}
        />
        <span
          style={{
            color: COLORS.text,
            fontSize: 20,
            fontWeight: 700,
            fontFamily: "system-ui",
            letterSpacing: -0.5,
          }}
        >
          Nexus
        </span>
        <span
          style={{
            color: COLORS.textMuted,
            fontSize: 12,
            marginLeft: "auto",
            marginRight: 20,
          }}
        >
          Music & Speakers
        </span>
      </div>

      {/* Chat area */}
      <div style={{ flex: 1, padding: 16, overflow: "hidden" }}>{children}</div>
    </div>
  );
};

// ─── Chat Bubbles ────────────────────────────────────────

const UserBubble: React.FC<{ text: string; startFrame: number }> = ({
  text,
  startFrame,
}) => (
  <FadeIn startFrame={startFrame}>
    <div
      style={{
        display: "flex",
        justifyContent: "flex-end",
        marginBottom: 10,
      }}
    >
      <div
        style={{
          background: COLORS.userBubble,
          color: COLORS.text,
          padding: "10px 16px",
          borderRadius: "18px 18px 4px 18px",
          maxWidth: 280,
          fontSize: 15,
          fontFamily: "system-ui",
          lineHeight: 1.4,
        }}
      >
        <TypeWriter text={text} startFrame={startFrame} speed={2} />
      </div>
    </div>
  </FadeIn>
);

const ThinkingBubble: React.FC<{ text: string; startFrame: number }> = ({
  text,
  startFrame,
}) => (
  <FadeIn startFrame={startFrame}>
    <div style={{ display: "flex", marginBottom: 8 }}>
      <div
        style={{
          background: COLORS.thinkingBubble,
          border: `1px solid ${COLORS.border}`,
          padding: "8px 12px",
          borderRadius: 12,
          maxWidth: 300,
          display: "flex",
          gap: 6,
          alignItems: "flex-start",
        }}
      >
        <span style={{ fontSize: 12 }}>🧠</span>
        <span
          style={{
            color: COLORS.textMuted,
            fontSize: 12,
            fontStyle: "italic",
            fontFamily: "system-ui",
            lineHeight: 1.4,
          }}
        >
          <TypeWriter text={text} startFrame={startFrame + 5} speed={2.5} />
        </span>
      </div>
    </div>
  </FadeIn>
);

const ToolCallBubble: React.FC<{
  name: string;
  args: string;
  startFrame: number;
}> = ({ name, args, startFrame }) => {
  const frame = useCurrentFrame();
  const isComplete = frame > startFrame + 30;

  return (
    <FadeIn startFrame={startFrame}>
      <div style={{ display: "flex", marginBottom: 6 }}>
        <div
          style={{
            background: COLORS.toolBubble,
            border: `1px solid ${COLORS.border}`,
            padding: "8px 12px",
            borderRadius: 10,
            maxWidth: 300,
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span style={{ fontSize: 10, color: isComplete ? COLORS.success : COLORS.secondary }}>
              {isComplete ? "✓" : "⟳"}
            </span>
            <span
              style={{
                color: COLORS.secondary,
                fontSize: 11,
                fontWeight: 700,
                fontFamily: "monospace",
                textTransform: "uppercase",
              }}
            >
              {name}
            </span>
          </div>
          <div
            style={{
              color: COLORS.textMuted,
              fontSize: 10,
              fontFamily: "monospace",
              marginTop: 4,
            }}
          >
            {args}
          </div>
        </div>
      </div>
    </FadeIn>
  );
};

const AssistantBubble: React.FC<{ text: string; startFrame: number }> = ({
  text,
  startFrame,
}) => (
  <FadeIn startFrame={startFrame}>
    <div style={{ display: "flex", marginBottom: 10 }}>
      <div
        style={{
          background: COLORS.assistantBubble,
          color: COLORS.text,
          padding: "10px 16px",
          borderRadius: "18px 18px 18px 4px",
          maxWidth: 300,
          fontSize: 14,
          fontFamily: "system-ui",
          lineHeight: 1.5,
        }}
      >
        <TypeWriter text={text} startFrame={startFrame} speed={2} />
      </div>
    </div>
  </FadeIn>
);

// ─── Now Playing Card ────────────────────────────────────

const NowPlayingCard: React.FC<{ startFrame: number }> = ({ startFrame }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const scale = spring({ frame: frame - startFrame, fps, config: { damping: 12 } });
  const pulse = Math.sin((frame - startFrame) * 0.1) * 0.02 + 1;

  return (
    <FadeIn startFrame={startFrame} style={{ marginBottom: 10 }}>
      <div
        style={{
          background: "linear-gradient(135deg, #1a1a3e 0%, #2d1b69 100%)",
          border: `1px solid ${COLORS.primary}`,
          borderRadius: 16,
          padding: 16,
          transform: `scale(${scale * pulse})`,
        }}
      >
        <div style={{ display: "flex", gap: 14, alignItems: "center" }}>
          <div
            style={{
              width: 56,
              height: 56,
              borderRadius: 8,
              background: `linear-gradient(135deg, ${COLORS.primary}, ${COLORS.secondary})`,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 24,
            }}
          >
            🎵
          </div>
          <div>
            <div
              style={{
                color: COLORS.text,
                fontSize: 14,
                fontWeight: 700,
                fontFamily: "system-ui",
              }}
            >
              Shape of You
            </div>
            <div style={{ color: COLORS.textMuted, fontSize: 12, fontFamily: "system-ui" }}>
              Ed Sheeran
            </div>
            <div
              style={{
                color: COLORS.success,
                fontSize: 11,
                fontFamily: "system-ui",
                marginTop: 4,
                display: "flex",
                alignItems: "center",
                gap: 4,
              }}
            >
              <span style={{ fontSize: 8 }}>●</span> Playing on Living Room
            </div>
          </div>
        </div>
        {/* Progress bar */}
        <div
          style={{
            height: 3,
            background: "#333",
            borderRadius: 2,
            marginTop: 12,
            overflow: "hidden",
          }}
        >
          <div
            style={{
              height: "100%",
              width: `${interpolate(frame, [startFrame, startFrame + 300], [0, 60], { extrapolateRight: "clamp" })}%`,
              background: `linear-gradient(90deg, ${COLORS.primary}, ${COLORS.secondary})`,
              borderRadius: 2,
            }}
          />
        </div>
      </div>
    </FadeIn>
  );
};

// ─── Feature Badges ──────────────────────────────────────

const FeatureBadge: React.FC<{
  label: string;
  icon: string;
  startFrame: number;
  index: number;
}> = ({ label, icon, startFrame, index }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const delay = index * 5;
  const s = spring({
    frame: frame - startFrame - delay,
    fps,
    config: { damping: 12, stiffness: 100 },
  });
  const scale = interpolate(s, [0, 1], [0.5, 1]);
  const opacity = interpolate(s, [0, 1], [0, 1]);

  return (
    <div
      style={{
        opacity,
        transform: `scale(${scale})`,
        background: "rgba(108, 99, 255, 0.15)",
        border: `1px solid rgba(108, 99, 255, 0.3)`,
        borderRadius: 20,
        padding: "6px 14px",
        display: "inline-flex",
        alignItems: "center",
        gap: 6,
      }}
    >
      <span style={{ fontSize: 14 }}>{icon}</span>
      <span style={{ color: COLORS.text, fontSize: 12, fontFamily: "system-ui", fontWeight: 500 }}>
        {label}
      </span>
    </div>
  );
};

// ─── Main Composition ────────────────────────────────────

export const NexusDemo: React.FC<{ layout?: "vertical" | "horizontal" }> = ({
  layout = "vertical",
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const isHorizontal = layout === "horizontal";

  // Title animation
  const titleOpacity = interpolate(frame, [0, 20, 50, 65], [0, 1, 1, 0], {
    extrapolateRight: "clamp",
  });
  const titleScale = spring({ frame, fps, config: { damping: 15 } });

  // CTA at end
  const ctaOpacity = interpolate(frame, [600, 630, 720, 750], [0, 1, 1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  // Overall fade out
  const fadeOut = interpolate(frame, [720, 750], [1, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <div
      style={{
        width: "100%",
        height: "100%",
        background: COLORS.bg,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        opacity: fadeOut,
      }}
    >
      {/* Ambient glow */}
      <div
        style={{
          position: "absolute",
          top: "30%",
          left: "50%",
          width: 600,
          height: 600,
          borderRadius: "50%",
          background: `radial-gradient(circle, rgba(108, 99, 255, 0.08) 0%, transparent 70%)`,
          transform: "translateX(-50%)",
        }}
      />

      {/* Title Card (frames 0-60) */}
      {frame < 65 && (
        <div
          style={{
            position: "absolute",
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            opacity: titleOpacity,
            transform: `scale(${interpolate(titleScale, [0, 1], [0.8, 1])})`,
          }}
        >
          <span
            style={{
              fontSize: 72,
              fontWeight: 800,
              fontFamily: "system-ui",
              background: `linear-gradient(135deg, ${COLORS.primary}, ${COLORS.secondary})`,
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
              letterSpacing: -2,
            }}
          >
            Nexus
          </span>
          <span
            style={{
              color: COLORS.textMuted,
              fontSize: 18,
              fontFamily: "system-ui",
              marginTop: 8,
            }}
          >
            Voice-controlled AI assistant
          </span>
        </div>
      )}

      {/* Phone with chat (frames 30+) */}
      {frame >= 30 && frame < 600 && (
        <PhoneFrame startFrame={30} layout={layout}>
          {/* Connection status */}
          <FadeIn startFrame={60}>
            <div
              style={{
                textAlign: "center",
                marginBottom: 16,
                padding: "6px 0",
              }}
            >
              <span style={{ color: COLORS.success, fontSize: 11, fontFamily: "system-ui" }}>
                ● Connected via mDNS
              </span>
            </div>
          </FadeIn>

          {/* First flow: play music */}
          {frame >= 90 && (
            <UserBubble text='play shape of you' startFrame={90} />
          )}
          {frame >= 130 && (
            <ThinkingBubble
              text="I'll search for that song and play it on your Sonos speaker."
              startFrame={130}
            />
          )}
          {frame >= 160 && (
            <ToolCallBubble
              name="radio_play"
              args='query: "Shape of You"'
              startFrame={160}
            />
          )}
          {frame >= 210 && (
            <AssistantBubble
              text='Shape of You is now playing on Living Room.'
              startFrame={210}
            />
          )}
          {frame >= 250 && <NowPlayingCard startFrame={250} />}

          {/* Second flow: volume control */}
          {frame >= 360 && (
            <UserBubble text="set volume to 30" startFrame={360} />
          )}
          {frame >= 390 && (
            <ToolCallBubble
              name="sonos_set_volume"
              args='device: "Living Room", volume: 30'
              startFrame={390}
            />
          )}
          {frame >= 430 && (
            <AssistantBubble text="Volume set to 30." startFrame={430} />
          )}
        </PhoneFrame>
      )}

      {/* Right side info panel for horizontal layout */}
      {isHorizontal && frame >= 60 && frame < 600 && (
        <div
          style={{
            position: "absolute",
            right: 120,
            top: "50%",
            transform: "translateY(-50%)",
            width: 550,
            display: "flex",
            flexDirection: "column",
            gap: 24,
          }}
        >
          <FadeIn startFrame={70}>
            <span
              style={{
                fontSize: 48,
                fontWeight: 800,
                fontFamily: "system-ui",
                background: `linear-gradient(135deg, ${COLORS.primary}, ${COLORS.secondary})`,
                WebkitBackgroundClip: "text",
                WebkitTextFillColor: "transparent",
                letterSpacing: -1,
              }}
            >
              Nexus
            </span>
          </FadeIn>
          <FadeIn startFrame={80}>
            <span style={{ color: COLORS.textMuted, fontSize: 18, fontFamily: "system-ui", lineHeight: 1.6 }}>
              Voice-controlled AI assistant that plays music on your Sonos speakers with a single command.
            </span>
          </FadeIn>
          {frame >= 530 && (
            <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
              {[
                { icon: "🎤", label: "Voice Control" },
                { icon: "🤖", label: "Agentic AI" },
                { icon: "🔄", label: "Auto-Recovery" },
                { icon: "🔊", label: "Sonos Integration" },
                { icon: "⚡", label: "8 LLM Slots" },
                { icon: "🔒", label: "TLS Encrypted" },
              ].map((f, i) => (
                <FeatureBadge key={f.label} icon={f.icon} label={f.label} startFrame={530} index={i} />
              ))}
            </div>
          )}
        </div>
      )}

      {/* Feature badges (frames 530-600) — vertical only */}
      {!isHorizontal && frame >= 530 && frame < 620 && (
        <div
          style={{
            position: "absolute",
            bottom: 200,
            display: "flex",
            flexWrap: "wrap",
            gap: 8,
            justifyContent: "center",
            maxWidth: 500,
            padding: "0 40px",
          }}
        >
          {[
            { icon: "🎤", label: "Voice Control" },
            { icon: "🤖", label: "Agentic AI" },
            { icon: "🔄", label: "Auto-Recovery" },
            { icon: "🔊", label: "Sonos Integration" },
            { icon: "⚡", label: "8 LLM Slots" },
            { icon: "🔒", label: "TLS Encrypted" },
          ].map((f, i) => (
            <FeatureBadge
              key={f.label}
              icon={f.icon}
              label={f.label}
              startFrame={530}
              index={i}
            />
          ))}
        </div>
      )}

      {/* CTA (frames 600-750) */}
      {frame >= 600 && (
        <div
          style={{
            position: "absolute",
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            opacity: ctaOpacity,
            gap: 16,
          }}
        >
          <span
            style={{
              fontSize: 52,
              fontWeight: 800,
              fontFamily: "system-ui",
              background: `linear-gradient(135deg, ${COLORS.primary}, ${COLORS.secondary})`,
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
              letterSpacing: -1,
            }}
          >
            Nexus
          </span>
          <span
            style={{
              color: COLORS.textMuted,
              fontSize: 16,
              fontFamily: "monospace",
            }}
          >
            github.com/rhythm493/nexus
          </span>
          <div
            style={{
              display: "flex",
              gap: 16,
              marginTop: 8,
            }}
          >
            <span style={{ color: COLORS.textMuted, fontSize: 13, fontFamily: "system-ui" }}>
              Go + Flutter + MCP + Liquidsoap
            </span>
          </div>
        </div>
      )}
    </div>
  );
};
