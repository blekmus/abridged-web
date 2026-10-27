const COLORS = [
  "#52dfff",
  "#34dcba",
  "#ef3e78",
  "#ffd556",
  "#bca6ff",
  "#fff1da",
];

let kidsCheerData: Promise<ArrayBuffer> | undefined;
let preparedKidsCheer: Promise<AudioBuffer | undefined> | undefined;

function loadKidsCheer(
  context: BaseAudioContext,
): Promise<AudioBuffer | undefined> {
  kidsCheerData ??= fetch("/audio/kids-yay.mp3?v=2")
    .then((response) => {
      if (!response.ok) throw new Error("Could not load the kids' cheer");
      return response.arrayBuffer();
    })
    .catch((error: unknown) => {
      kidsCheerData = undefined;
      throw error;
    });
  return kidsCheerData
    .then((data) => context.decodeAudioData(data.slice(0)))
    .catch(() => undefined);
}

/** Decode ahead of the click without creating a live or audible audio context. */
export function preloadSurpriseAudio(): Promise<AudioBuffer | undefined> {
  if (preparedKidsCheer) return preparedKidsCheer;
  try {
    const decoder = new OfflineAudioContext(1, 1, 44100);
    preparedKidsCheer = loadKidsCheer(decoder).then((buffer) => {
      if (!buffer) preparedKidsCheer = undefined;
      return buffer;
    });
    return preparedKidsCheer;
  } catch {
    return Promise.resolve(undefined);
  }
}

export function createSurpriseSound(onCheerStart?: () => void) {
  let context: AudioContext | undefined;
  let closed = false;
  let cheerStartTimer: number | undefined;
  let activeVoices = 0;
  let pendingClips = 0;
  let finishRequested = false;
  try {
    context = new AudioContext();
    void context.resume().catch(() => {});
  } catch {
    // Audio is optional; the physical release still has visual feedback.
  }
  const audio = context;
  const cheerReady = audio
    ? (preparedKidsCheer ?? Promise.resolve(undefined)).then(
        (buffer) => buffer ?? loadKidsCheer(audio),
      )
    : Promise.resolve(undefined);
  const dispose = () => {
    if (closed) return;
    closed = true;
    window.clearTimeout(cheerStartTimer);
    if (audio && audio.state !== "closed") void audio.close().catch(() => {});
  };
  const finishIfIdle = () => {
    if (finishRequested && activeVoices === 0 && pendingClips === 0) dispose();
  };
  const trackVoice = (node: AudioScheduledSourceNode, cleanup: () => void) => {
    activeVoices += 1;
    node.onended = () => {
      cleanup();
      activeVoices -= 1;
      finishIfIdle();
    };
  };
  const tone = (
    frequency: number,
    endFrequency: number,
    start: number,
    duration: number,
    volume: number,
    type: OscillatorType = "sine",
  ) => {
    if (!audio || closed || audio.state !== "running") return;
    const oscillator = audio.createOscillator();
    const gain = audio.createGain();
    oscillator.type = type;
    oscillator.frequency.setValueAtTime(frequency, start);
    oscillator.frequency.exponentialRampToValueAtTime(
      endFrequency,
      start + duration,
    );
    gain.gain.setValueAtTime(0.0001, start);
    gain.gain.exponentialRampToValueAtTime(
      volume,
      start + Math.min(0.018, duration / 4),
    );
    gain.gain.exponentialRampToValueAtTime(0.0001, start + duration);
    oscillator.connect(gain).connect(audio.destination);
    oscillator.start(start);
    oscillator.stop(start + duration + 0.02);
    trackVoice(oscillator, () => {
      oscillator.disconnect();
      gain.disconnect();
    });
  };
  return {
    tension() {
      if (!audio || closed) return;
      const now = audio.currentTime;
      tone(74, 125, now, 0.22, 0.09, "triangle");
      tone(140, 390, now + 0.12, 0.48, 0.035, "triangle");
      tone(310, 660, now + 0.25, 0.35, 0.018);
    },
    release() {
      if (!audio || closed) return;
      const now = audio.currentTime;
      pendingClips += 1;
      void cheerReady
        .then((buffer) => {
          if (!buffer || closed || audio.state !== "running") return;
          const cheer = audio.createBufferSource();
          const cheerGain = audio.createGain();
          cheer.buffer = buffer;
          cheerGain.gain.value = 0.55;
          cheer.connect(cheerGain).connect(audio.destination);
          const startAt = Math.max(now + 0.25, audio.currentTime);
          cheer.start(startAt);
          if (onCheerStart) {
            cheerStartTimer = window.setTimeout(
              () => {
                if (!closed) onCheerStart();
              },
              Math.max(0, (startAt - audio.currentTime) * 1000),
            );
          }
          trackVoice(cheer, () => {
            cheer.disconnect();
            cheerGain.disconnect();
          });
        })
        .catch(() => {})
        .finally(() => {
          pendingClips -= 1;
          finishIfIdle();
        });
      tone(320, 54, now, 0.16, 0.18, "triangle");
      // A short filtered air burst and an original ascending glass chime.
      const buffer = audio.createBuffer(
        1,
        Math.ceil(audio.sampleRate * 0.22),
        audio.sampleRate,
      );
      const samples = buffer.getChannelData(0);
      for (let i = 0; i < samples.length; i++)
        samples[i] = (Math.random() * 2 - 1) * (1 - i / samples.length) ** 3;
      const source = audio.createBufferSource();
      const filter = audio.createBiquadFilter();
      const gain = audio.createGain();
      source.buffer = buffer;
      filter.type = "lowpass";
      filter.frequency.setValueAtTime(5200, now);
      filter.frequency.exponentialRampToValueAtTime(420, now + 0.22);
      gain.gain.value = 0.15;
      source.connect(filter).connect(gain).connect(audio.destination);
      source.start(now);
      trackVoice(source, () => {
        source.disconnect();
        filter.disconnect();
        gain.disconnect();
      });
      [659.25, 987.77, 1318.51, 1975.53].forEach((frequency, index) => {
        tone(
          frequency,
          frequency * 0.998,
          now + 0.035 + index * 0.055,
          0.72 - index * 0.07,
          0.055 / (1 + index * 0.3),
        );
      });
      finishRequested = true;
      finishIfIdle();
    },
    finish() {
      finishRequested = true;
      finishIfIdle();
    },
    dispose,
  };
}

export function launchCornerConfetti(): () => void {
  const canvas = document.createElement("canvas");
  canvas.className = "surprise-confetti";
  canvas.setAttribute("aria-hidden", "true");
  document.body.append(canvas);
  const context = canvas.getContext("2d");
  if (!context) {
    canvas.remove();
    return () => {};
  }
  let width = window.innerWidth;
  let height = window.innerHeight;
  const resize = () => {
    width = window.innerWidth;
    height = window.innerHeight;
    const ratio = Math.min(window.devicePixelRatio || 1, 2);
    canvas.width = width * ratio;
    canvas.height = height * ratio;
    canvas.style.width = `${width}px`;
    canvas.style.height = `${height}px`;
    context.setTransform(ratio, 0, 0, ratio, 0, 0);
  };
  resize();
  window.addEventListener("resize", resize);
  const count = width < 620 ? 65 : 105;
  const particles = [-1, 1].flatMap((side) =>
    Array.from({ length: count }, (_, index) => {
      const speed = Math.min(width * 0.6, 760) * (0.55 + Math.random() * 0.7);
      const angle = ((0.38 + Math.random() * 0.72) * Math.PI) / 2;
      const horizontalSpeed = Math.cos(angle) * speed;
      const upwardSpeed = height * 0.95 + Math.random() * height * 0.65;
      const launchSpeed = Math.hypot(horizontalSpeed, upwardSpeed);
      const launchAngle = Math.atan2(upwardSpeed, horizontalSpeed) * 0.9;
      return {
        x: side < 0 ? -8 : width + 8,
        y: height + 10,
        vx: -side * Math.cos(launchAngle) * launchSpeed,
        vy: -Math.sin(launchAngle) * launchSpeed,
        size: 5 + Math.random() * 6,
        rotation: Math.random() * Math.PI * 2,
        spin: (Math.random() - 0.5) * 13,
        wobble: Math.random() * Math.PI * 2,
        delay: (index % 3) * 0.045,
        color: COLORS[index % COLORS.length] ?? "#ffd556",
        ribbon: index % 6 === 0,
      };
    }),
  );
  let frame = 0;
  let elapsed = 0;
  let previous = performance.now();
  let disposed = false;
  const dispose = () => {
    if (disposed) return;
    disposed = true;
    cancelAnimationFrame(frame);
    window.removeEventListener("resize", resize);
    canvas.remove();
  };
  const draw = (now: number) => {
    if (disposed) return;
    const dt = Math.min((now - previous) / 1000, 0.035);
    previous = now;
    elapsed += dt;
    context.clearRect(0, 0, width, height);
    for (const particle of particles) {
      if (elapsed < particle.delay) continue;
      particle.vx *= Math.exp(-dt * 0.5);
      particle.vy += 660 * dt;
      particle.x += particle.vx * dt;
      particle.y += particle.vy * dt;
      particle.rotation += particle.spin * dt;
      particle.wobble += dt * 7;
      const flutter = Math.cos(particle.wobble);
      context.save();
      context.translate(particle.x + Math.sin(particle.wobble) * 5, particle.y);
      context.rotate(particle.rotation);
      context.scale(1, Math.max(0.15, Math.abs(flutter)));
      context.globalAlpha = Math.min(1, (5.2 - elapsed) / 0.65);
      context.fillStyle = particle.color;
      if (particle.ribbon) {
        context.beginPath();
        context.moveTo(-particle.size, -2);
        context.bezierCurveTo(-3, -8, 3, 8, particle.size, 2);
        context.lineTo(particle.size, 5);
        context.bezierCurveTo(3, 11, -3, -5, -particle.size, 1);
        context.fill();
      } else
        context.fillRect(
          -particle.size / 2,
          -particle.size / 3,
          particle.size,
          particle.size * 0.65,
        );
      context.restore();
    }
    if (elapsed < 5.2) frame = requestAnimationFrame(draw);
    else dispose();
  };
  frame = requestAnimationFrame(draw);
  return dispose;
}

let stopPreviousCelebration: (() => void) | undefined;

/** The celebration outlives the welcome panel, but a fresh run replaces it. */
export function startSurpriseCelebration(
  onCheerStart: () => void,
  withConfetti: boolean,
) {
  stopPreviousCelebration?.();
  const sound = createSurpriseSound(onCheerStart);
  let stopConfetti: (() => void) | undefined;
  stopPreviousCelebration = () => {
    stopConfetti?.();
    sound.dispose();
  };
  return {
    tension: sound.tension,
    release() {
      sound.release();
      if (withConfetti) stopConfetti = launchCornerConfetti();
    },
    finish: sound.finish,
  };
}
