#version 330

in vec3 fragPosition;
out vec4 finalColor;

uniform sampler2D texture0;
uniform float tileSize;

uniform vec3 cameraPos;
uniform vec3 fogColor;
uniform float fogStart;
uniform float fogEnd;

uniform vec3 lightDir;
uniform vec3 lightColor;
uniform vec3 ambientColor;

void main() {
    vec2 uv = fragPosition.xz / tileSize;
    vec4 base = texture(texture0, uv);

    vec3 N = vec3(0.0, 1.0, 0.0);
    vec3 L = normalize(-lightDir);
    float diff = max(dot(N, L), 0.0);
    vec3 lit = base.rgb * (ambientColor + diff * lightColor);

    float dist = length(fragPosition - cameraPos);
    float fog = clamp((dist - fogStart) / max(fogEnd - fogStart, 0.0001), 0.0, 1.0);
    fog = fog * fog;
    vec3 rgb = mix(lit, fogColor, fog);
    finalColor = vec4(rgb, base.a);
}
