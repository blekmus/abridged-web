import * as THREE from "three";
import { RoomEnvironment } from "three/addons/environments/RoomEnvironment.js";
import {
  CSS3DObject,
  CSS3DRenderer,
} from "three/addons/renderers/CSS3DRenderer.js";

import { softenLowerRim } from "./softenLowerRim";
import { createSurpriseBodyGeometry } from "./surpriseBodyGeometry";

export type SurpriseStage = "lifting" | "straining" | "released" | "settled";

type Options = {
  host: HTMLElement;
  source: HTMLButtonElement;
  onStart: () => void;
  onStage: (stage: SurpriseStage) => void;
  onRelease: () => void;
  onUnavailable: () => void;
  deferStart?: boolean;
  startFlush?: boolean;
};

const smooth = (from: number, to: number, value: number) => {
  const t = THREE.MathUtils.clamp((value - from) / (to - from), 0, 1);
  return t * t * (3 - 2 * t);
};

let sharedRenderer: THREE.WebGLRenderer | undefined;
let sharedEnvironment: THREE.WebGLRenderTarget | undefined;
let warmScene: { dispose: () => void; warmup: () => Promise<void> } | undefined;
let warmHost: HTMLDivElement | undefined;

/** Keep both compiled material variants alive in the same context used by the visible scene. */
export async function prewarmEntrySurprise(): Promise<void> {
  if (warmScene) return;
  const host = document.createElement("div");
  host.setAttribute("aria-hidden", "true");
  host.inert = true;
  Object.assign(host.style, {
    position: "fixed",
    left: "-10000px",
    top: "0",
    visibility: "hidden",
    pointerEvents: "none",
  });
  const source = document.createElement("button");
  source.type = "button";
  const phone = window.innerWidth <= 620;
  source.style.width = `${Math.max(160, Math.min(740, window.innerWidth - (phone ? 32 : 64) - 50))}px`;
  source.style.height = phone ? "80px" : "66px";
  host.append(source);
  document.body.append(host);
  warmHost = host;
  try {
    warmScene = createEntrySurprise({
      host,
      source,
      deferStart: true,
      onStart() {},
      onStage() {},
      onRelease() {},
      onUnavailable() {},
    });
    await warmScene.warmup();
  } catch (error) {
    disposeEntrySurprisePreparation();
    throw error;
  }
}

export function disposeEntrySurprisePreparation(): void {
  warmScene?.dispose();
  warmScene = undefined;
  warmHost?.remove();
  warmHost = undefined;
  sharedEnvironment?.dispose();
  sharedEnvironment = undefined;
  sharedRenderer?.dispose();
  sharedRenderer?.forceContextLoss();
  sharedRenderer = undefined;
}

export function createEntrySurprise(options: Options) {
  const { host, source } = options;
  const width = source.offsetWidth;
  const height = source.offsetHeight;
  const canvasWidth = width + 192;
  const canvasHeight = height + 192;
  sharedRenderer ??= new THREE.WebGLRenderer({
    alpha: true,
    antialias: true,
    powerPreference: "low-power",
  });
  const renderer = sharedRenderer;
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  renderer.setSize(canvasWidth, canvasHeight);
  renderer.setClearColor(0x101010, 0);
  renderer.toneMapping = THREE.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.05;
  renderer.domElement.className = "surprise-webgl";
  host.append(renderer.domElement);

  const cssRenderer = new CSS3DRenderer();
  cssRenderer.setSize(canvasWidth, canvasHeight);
  cssRenderer.domElement.className = "surprise-label-layer";
  cssRenderer.domElement.setAttribute("aria-hidden", "true");
  host.append(cssRenderer.domElement);

  const scene = new THREE.Scene();
  // A real dark transmission background prevents transparent pixels from refracting white.
  scene.background = new THREE.Color(0x101010);
  const distance = 900;
  const camera = new THREE.PerspectiveCamera(
    (2 * Math.atan(canvasHeight / (2 * distance)) * 180) / Math.PI,
    canvasWidth / canvasHeight,
    0.1,
    2400,
  );
  camera.position.z = distance;
  if (!sharedEnvironment) {
    const environment = new RoomEnvironment();
    const pmrem = new THREE.PMREMGenerator(renderer);
    sharedEnvironment = pmrem.fromScene(environment, 0.045);
    environment.dispose();
    pmrem.dispose();
  }
  scene.environment = sharedEnvironment.texture;
  scene.environmentRotation.set(0.6, 0.8, 0.2);
  scene.add(new THREE.AmbientLight(0xdce9ff, 0.8));
  const key = new THREE.DirectionalLight(0xfff4e1, 3.5);
  key.position.set(-width * 0.4, 160, 280);
  scene.add(key);
  const rim = new THREE.PointLight(0x6adfff, 24000, 1000, 2);
  rim.position.set(width * 0.48, -60, 100);
  scene.add(rim);
  const warm = new THREE.PointLight(0xffd69a, 18000, 1000, 2);
  warm.position.set(-width * 0.4, 90, 150);
  scene.add(warm);

  // The lifted entry leaves a plain surface; only the diffuse shadow remains below it.
  const shadowMaterial = new THREE.ShaderMaterial({
    transparent: true,
    depthWrite: false,
    uniforms: {
      uVisibility: { value: 0 },
      uSize: { value: new THREE.Vector2(width, height) },
    },
    vertexShader:
      "varying vec2 vUv; void main(){vUv=uv;gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.);}",
    fragmentShader: `
      uniform float uVisibility;
      uniform vec2 uSize;
      varying vec2 vUv;
      void main(){
        vec2 p = (vUv - .5) * (uSize + vec2(192.,192.));
        p.y += 8.;
        vec2 q = abs(p) - uSize * .5 + 12.;
        float d = length(max(q, 0.)) + min(max(q.x, q.y), 0.) - 12.;
        float outside = max(d, 0.);
        float alpha = .065 * exp(-outside * outside / (2. * 34. * 34.)) * uVisibility;
        gl_FragColor=vec4(0.,0.,0.,alpha);
      }
    `,
  });
  const shadowGeometry = new THREE.PlaneGeometry(width + 192, height + 192);
  const shadow = new THREE.Mesh(shadowGeometry, shadowMaterial);
  shadow.position.z = 0.1;
  shadow.renderOrder = 1;
  scene.add(shadow);

  const group = new THREE.Group();
  scene.add(group);
  const bodyMaterial = new THREE.MeshPhysicalMaterial({
    color: options.startFlush ? 0x101010 : 0x121211,
    metalness: 0.82,
    roughness: 0.7,
    clearcoat: 0.3,
    clearcoatRoughness: 0.12,
    transmission: 0,
    thickness: 10,
    ior: 1.46,
    iridescence: 0,
    iridescenceIOR: 1.35,
    dispersion: 0.18,
    envMapIntensity: 0.65,
  });
  const lowerRimSoftness = softenLowerRim(bodyMaterial, height);
  const bodyGeometry = createSurpriseBodyGeometry(width, height);
  const body = new THREE.Mesh(bodyGeometry, bodyMaterial);
  body.renderOrder = 2;
  group.add(body);
  const label = document.createElement("div");
  label.className = "search-option surprise-label";
  label.style.width = `${width}px`;
  label.style.height = `${height}px`;
  for (const child of source.children) label.append(child.cloneNode(true));
  const labelObject = new CSS3DObject(label);
  labelObject.position.z = 5.5;
  group.add(labelObject);

  const metal = new THREE.Color(options.startFlush ? 0x101010 : 0x121211);
  const baseZ = options.startFlush ? -5 : -6;
  const travelScale = 0.55;
  const returnFraction = 0.5;
  const restingHeight = 15 * travelScale * (1 - returnFraction);
  const glass = new THREE.Color(0xaebfc8);
  let frame = 0;
  let disposed = false;
  let released = false;
  let settled = false;
  let lastStage: SurpriseStage | undefined;
  let elapsed = 0;
  let previous = 0;
  let tiltX = 0;
  let tiltY = 0;
  let currentTiltX = 0;
  let currentTiltY = 0;
  const stage = (value: SurpriseStage) => {
    if (lastStage === value) return;
    lastStage = value;
    options.onStage(value);
  };
  const render = () => {
    renderer.render(scene, camera);
    cssRenderer.render(scene, camera);
  };
  const draw = (now: number) => {
    if (disposed) return;
    frame = 0;
    if (!previous) {
      previous = now;
      options.onStart();
    }
    const dt = Math.min((now - previous) / 1000, 0.05);
    previous = now;
    elapsed += dt;
    const lift = smooth(0.06, 0.34, elapsed);
    const strain = smooth(0.34, 0.6, elapsed);
    const release = smooth(0.6, 0.83, elapsed);
    const glassOptics = smooth(0.6, 0.82, elapsed);
    const reveal = smooth(0.82, 1.24, elapsed);
    const softHighlight =
      smooth(0.82, 1.03, elapsed) * (1 - smooth(1.03, 1.6, elapsed));
    const springTime = Math.max(0, elapsed - 0.6);
    const spring =
      1 - Math.exp(-springTime * 4.8) * Math.cos(springTime * 12.5);
    const returnToRest = smooth(0.24, 0.68, springTime);
    const tremor = Math.sin(elapsed * 74) * strain * (1 - release);
    group.position.z =
      baseZ + (lift * 8 + strain * 13 + spring * 34) * travelScale;
    // Return halfway from the old floating height, preserving the spring's settling motion.
    group.position.y =
      elapsed >= 2.4
        ? restingHeight
        : (lift * 2 +
            strain * 2 +
            spring * 11 -
            returnToRest * 15 * returnFraction) *
          travelScale;
    group.position.x = tremor * 0.35;
    group.scale.set(1 - release * 0.025, 1 + release * 0.04, 1);
    const hoverEnabled = elapsed >= 0.6;
    currentTiltX = THREE.MathUtils.damp(
      currentTiltX,
      hoverEnabled ? tiltX : 0,
      8,
      dt,
    );
    currentTiltY = THREE.MathUtils.damp(
      currentTiltY,
      hoverEnabled ? tiltY : 0,
      8,
      dt,
    );
    group.rotation.x =
      -lift * 0.035 -
      strain * 0.035 +
      release * 0.025 +
      Math.sin(springTime * 11) * Math.exp(-springTime * 4) * release * 0.17 +
      currentTiltX;
    group.rotation.y = tremor * 0.0015 + currentTiltY;
    group.rotation.z =
      Math.sin(springTime * 10) * Math.exp(-springTime * 3.5) * release * 0.012;
    bodyMaterial.color.copy(metal).lerp(glass, reveal);
    // Establish transparency before brightening the tint, avoiding an opaque white flash.
    bodyMaterial.metalness = THREE.MathUtils.lerp(0.82, 0, glassOptics);
    bodyMaterial.roughness = THREE.MathUtils.lerp(0.7, 0.075, reveal);
    lowerRimSoftness.value = reveal;
    bodyMaterial.transmission = glassOptics * 0.98;
    bodyMaterial.clearcoat = THREE.MathUtils.lerp(0.3, 1, reveal);
    bodyMaterial.iridescence = glassOptics * 0.18;
    bodyMaterial.envMapIntensity = 0.65 + softHighlight * 0.04;
    // A steady, diffuse shadow: only its initial fade follows the lift, never the spring.
    const shadowVisibility = shadowMaterial.uniforms.uVisibility;
    if (shadowVisibility) shadowVisibility.value = smooth(0.08, 0.6, elapsed);
    if (elapsed < 0.34) stage("lifting");
    else if (elapsed < 0.6) stage("straining");
    else if (!released) {
      released = true;
      stage("released");
      options.onRelease();
    }
    if (elapsed > 2.4 && !settled) {
      settled = true;
      stage("settled");
    }
    render();
    if (
      !settled ||
      Math.abs(currentTiltX - tiltX) + Math.abs(currentTiltY - tiltY) > 0.0001
    )
      frame = requestAnimationFrame(draw);
  };
  const wake = () => {
    if (!frame && !disposed) {
      previous = performance.now();
      frame = requestAnimationFrame(draw);
    }
  };
  const onPointerMove = (event: PointerEvent) => {
    if (event.pointerType === "touch") return;
    const rect = source.getBoundingClientRect();
    tiltX =
      THREE.MathUtils.clamp(
        (event.clientY - rect.top) / rect.height - 0.5,
        -0.5,
        0.5,
      ) * 0.08;
    tiltY =
      THREE.MathUtils.clamp(
        (event.clientX - rect.left) / rect.width - 0.5,
        -0.5,
        0.5,
      ) * 0.08;
    if (released) wake();
  };
  const onPointerLeave = () => {
    tiltX = 0;
    tiltY = 0;
    if (released) wake();
  };
  const onContextLost = (event: Event) => {
    event.preventDefault();
    options.onUnavailable();
  };
  source.addEventListener("pointermove", onPointerMove);
  source.addEventListener("pointerleave", onPointerLeave);
  renderer.domElement.addEventListener("webglcontextlost", onContextLost);

  group.position.z = baseZ;
  if (!options.deferStart) {
    // The page's warm scene keeps these programs compiled in this shared context.
    bodyMaterial.transmission = 0.98;
    bodyMaterial.iridescence = 0.18;
    render();
    bodyMaterial.transmission = 0;
    bodyMaterial.iridescence = 0;
    render();
    frame = requestAnimationFrame(draw);
  }
  return {
    present() {
      if (!disposed) render();
    },
    start() {
      if (!disposed && !frame && !previous) frame = requestAnimationFrame(draw);
    },
    async warmup() {
      // KHR_parallel_shader_compile lets supported drivers prepare without blocking input.
      bodyMaterial.transmission = 0.98;
      bodyMaterial.iridescence = 0.18;
      await renderer.compileAsync(scene, camera);
      render();
      bodyMaterial.transmission = 0;
      bodyMaterial.iridescence = 0;
      await renderer.compileAsync(scene, camera);
      render();
    },
    dispose() {
      if (disposed) return;
      disposed = true;
      cancelAnimationFrame(frame);
      source.removeEventListener("pointermove", onPointerMove);
      source.removeEventListener("pointerleave", onPointerLeave);
      renderer.domElement.removeEventListener(
        "webglcontextlost",
        onContextLost,
      );
      for (const geometry of [shadowGeometry, bodyGeometry]) geometry.dispose();
      for (const material of [shadowMaterial, bodyMaterial]) material.dispose();
      label.remove();
      host.replaceChildren();
    },
  };
}
