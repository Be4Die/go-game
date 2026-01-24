#version 330

in vec3 fragPosition;
in vec2 fragTexCoord;
in vec3 fragNormal;

out vec4 finalColor;

uniform sampler2D texture0;

uniform vec3 cameraPos;

uniform vec3 lightDir;
uniform vec3 lightColor;
uniform vec3 ambientColor;
uniform float specStrength;
uniform float shininess;

uniform vec3 fogColor;
uniform float fogStart;
uniform float fogEnd;

void main() {
    vec4 base = texture(texture0, fragTexCoord);
    vec3 albedo = base.rgb;

    vec3 N = normalize(fragNormal);
    vec3 L = normalize(-lightDir);
    float diff = max(dot(N, L), 0.0);

    vec3 V = normalize(cameraPos - fragPosition);
    vec3 H = normalize(L + V);
    float spec = pow(max(dot(N, H), 0.0), shininess) * specStrength;

    vec3 lit = albedo * (ambientColor + diff * lightColor) + spec * lightColor;

    float dist = length(fragPosition - cameraPos);
    float fog = clamp((dist - fogStart) / max(fogEnd - fogStart, 0.0001), 0.0, 1.0);
    fog = fog * fog;
    vec3 rgb = mix(lit, fogColor, fog);

    finalColor = vec4(rgb, base.a);
}

