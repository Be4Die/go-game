#version 330

in vec3 fragPosition;
in vec2 fragTexCoord;

out vec4 finalColor;

uniform sampler2D texture0;

uniform vec3 cameraPos;
uniform vec3 fogColor;
uniform float fogStart;
uniform float fogEnd;

void main() {
    vec4 base = texture(texture0, fragTexCoord);
    float dist = length(fragPosition - cameraPos);
    float fog = clamp((dist - fogStart) / max(fogEnd - fogStart, 0.0001), 0.0, 1.0);
    vec3 rgb = mix(base.rgb, fogColor, fog);
    finalColor = vec4(rgb, base.a);
}

