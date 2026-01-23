#version 330

in vec3 fragPosition;
out vec4 finalColor;

uniform sampler2D texture0;
uniform float tileSize;

void main() {
    vec2 uv = fragPosition.xz / tileSize;
    finalColor = texture(texture0, uv);
}
