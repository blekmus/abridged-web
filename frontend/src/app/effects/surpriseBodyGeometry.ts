import * as THREE from "three";
import { mergeVertices } from "three/addons/utils/BufferGeometryUtils.js";

/** One continuously rounded surface, rather than separately shaded cap and side rims. */
export function createSurpriseBodyGeometry(
  width: number,
  height: number,
): THREE.BufferGeometry {
  const outline = new THREE.Shape();
  const halfWidth = width / 2 - 2;
  const halfHeight = height / 2 - 2;
  const radius = Math.min(12, halfHeight / 2);
  outline.moveTo(-halfWidth + radius, -halfHeight);
  outline.lineTo(halfWidth - radius, -halfHeight);
  outline.quadraticCurveTo(
    halfWidth,
    -halfHeight,
    halfWidth,
    -halfHeight + radius,
  );
  outline.lineTo(halfWidth, halfHeight - radius);
  outline.quadraticCurveTo(
    halfWidth,
    halfHeight,
    halfWidth - radius,
    halfHeight,
  );
  outline.lineTo(-halfWidth + radius, halfHeight);
  outline.quadraticCurveTo(
    -halfWidth,
    halfHeight,
    -halfWidth,
    halfHeight - radius,
  );
  outline.lineTo(-halfWidth, -halfHeight + radius);
  outline.quadraticCurveTo(
    -halfWidth,
    -halfHeight,
    -halfWidth + radius,
    -halfHeight,
  );
  const extrusion = new THREE.ExtrudeGeometry(outline, {
    depth: 6,
    bevelEnabled: true,
    bevelSegments: 12,
    steps: 1,
    bevelSize: 2,
    bevelThickness: 2,
    curveSegments: 12,
  });
  extrusion.translate(0, 0, -3);
  // The untextured body needs shared vertices across the cap/bevel seam, not separate UV islands.
  extrusion.deleteAttribute("normal");
  extrusion.deleteAttribute("uv");
  const geometry = mergeVertices(extrusion);
  geometry.computeVertexNormals();
  extrusion.dispose();
  return geometry;
}
