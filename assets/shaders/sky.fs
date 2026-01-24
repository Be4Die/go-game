#version 330

in vec3 fragPosition;
out vec4 finalColor;

uniform vec3 cameraPos;
uniform vec3 topColor;
uniform vec3 horizonColor;

uniform vec3 sunDir;
uniform vec3 sunColor;

void main() {
    vec3 dir = normalize(fragPosition - cameraPos);
    float t = clamp(dir.y * 0.5 + 0.5, 0.0, 1.0);

    vec3 sky = mix(horizonColor, topColor, pow(t, 1.4));

    float s = max(dot(dir, normalize(sunDir)), 0.0);
    float sunCore = pow(s, 800.0);
    float sunGlow = pow(s, 40.0) * 0.15;
    sky += sunColor * (sunCore + sunGlow);

    finalColor = vec4(sky, 1.0);
}

