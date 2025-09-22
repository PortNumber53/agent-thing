# Arch Linux based development environment
FROM archlinux:latest

# Update the system and install base development tools
RUN pacman -Syu --noconfirm && \
    pacman -S --noconfirm base-devel git openssh coreutils

# Create a non-root user for development
RUN useradd -m -G wheel -s /bin/bash developer && \
    echo "developer:developer" | chpasswd

# Allow wheel group to have sudo privileges
RUN echo "%wheel ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

# Set the user and working directory
USER developer
WORKDIR /home/developer

# Keep the container running
CMD ["tail", "-f", "/dev/null"]
