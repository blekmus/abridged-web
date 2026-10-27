import type { MeshPhysicalMaterial } from "three";

/** Broaden and feather the lower rim while leaving the upper glass optics intact. */
export function softenLowerRim(material: MeshPhysicalMaterial, height: number) {
  const amount = { value: 0 };
  const bottom = { value: -height / 2 };
  material.transparent = true;
  material.customProgramCacheKey = () => "abridged-lower-rim-v1";
  material.onBeforeCompile = (shader) => {
    shader.uniforms.uLowerRimAmount = amount;
    shader.uniforms.uBodyBottom = bottom;
    shader.vertexShader = shader.vertexShader
      .replace(
        "#include <common>",
        "#include <common>\nvarying float vBodyHeight;",
      )
      .replace(
        "#include <begin_vertex>",
        "#include <begin_vertex>\nvBodyHeight = position.y;",
      );
    shader.fragmentShader = shader.fragmentShader
      .replace(
        "#include <common>",
        `#include <common>
        varying float vBodyHeight;
        uniform float uLowerRimAmount;
        uniform float uBodyBottom;
      `,
      )
      .replace(
        "#include <lights_physical_fragment>",
        `#include <lights_physical_fragment>
        float lowerRim = 1. - smoothstep(uBodyBottom + 2., uBodyBottom + 14., vBodyHeight);
        float lowerSoftness = lowerRim * uLowerRimAmount;
        material.roughness = mix(material.roughness, max(material.roughness, .2), lowerSoftness);
        #ifdef USE_CLEARCOAT
          material.clearcoatRoughness = mix(material.clearcoatRoughness, max(material.clearcoatRoughness, .32), lowerSoftness);
        #endif
      `,
      )
      .replace(
        "#include <opaque_fragment>",
        `#include <opaque_fragment>
        float bottomCoverage = smoothstep(uBodyBottom, uBodyBottom + 1.4, vBodyHeight);
        gl_FragColor.a *= mix(1., bottomCoverage, uLowerRimAmount);
      `,
      );
  };
  return amount;
}
