// Dog Photo Management Module
class DogPhotoManager {
    constructor() {
        this.maxSizeMB = 10;
        this.allowedTypes = ['image/jpeg', 'image/png'];
        this.selectedFile = null;
        this.currentDogId = null;
        this.uploadInProgress = false;
    }

    /**
     * Validate file type and size
     */
    validateFile(file) {
        // Check file type
        if (!this.allowedTypes.includes(file.type)) {
            throw new Error('Nur JPEG und PNG Dateien sind erlaubt');
        }

        // Check file size
        const sizeMB = file.size / (1024 * 1024);
        if (sizeMB > this.maxSizeMB) {
            throw new Error(`Datei zu groß. Maximum: ${this.maxSizeMB}MB`);
        }

        return true;
    }

    /**
     * Preview file in the preview element
     */
    previewFile(file, previewElementId) {
        return new Promise((resolve, reject) => {
            try {
                this.validateFile(file);

                const reader = new FileReader();
                reader.onload = (e) => {
                    const previewImg = document.getElementById(previewElementId);
                    if (previewImg) {
                        previewImg.src = e.target.result;
                        previewImg.style.display = 'block';
                    }

                    // Show preview container and hide prompt
                    const photoPreview = document.getElementById('photo-preview');
                    const uploadPrompt = document.querySelector('.upload-prompt');

                    if (photoPreview) {
                        photoPreview.classList.remove('hidden');
                    }
                    if (uploadPrompt) {
                        uploadPrompt.style.display = 'none';
                    }

                    this.selectedFile = file;
                    resolve(e.target.result);
                };

                reader.onerror = () => {
                    reject(new Error('Fehler beim Lesen der Datei'));
                };

                reader.readAsDataURL(file);
            } catch (error) {
                reject(error);
            }
        });
    }

    /**
     * Clear preview
     */
    clearPreview() {
        const previewImg = document.getElementById('preview-img');
        const photoPreview = document.getElementById('photo-preview');
        const uploadPrompt = document.querySelector('.upload-prompt');
        const fileInput = document.getElementById('dog-photo');

        if (previewImg) {
            previewImg.src = '';
            previewImg.style.display = 'none';
        }
        if (photoPreview) {
            photoPreview.classList.add('hidden');
        }
        if (uploadPrompt) {
            uploadPrompt.style.display = 'flex';
        }
        if (fileInput) {
            fileInput.value = '';
        }

        this.selectedFile = null;
    }

    /**
     * Upload photo for a dog
     */
    async uploadPhoto(dogId, file) {
        if (this.uploadInProgress) {
            throw new Error('Upload läuft bereits');
        }

        try {
            this.uploadInProgress = true;
            this.validateFile(file);

            // Show progress indicator
            this.showProgress();

            const response = await api.uploadDogPhoto(dogId, file);

            this.hideProgress();
            this.uploadInProgress = false;

            return response;
        } catch (error) {
            this.hideProgress();
            this.uploadInProgress = false;
            throw error;
        }
    }

    /**
     * Upload current selected file
     */
    async uploadSelectedFile(dogId) {
        if (!this.selectedFile) {
            throw new Error('Keine Datei ausgewählt');
        }

        return this.uploadPhoto(dogId, this.selectedFile);
    }

    /**
     * Setup drag and drop for an element
     */
    setupDragDrop(zoneId, onFileSelected) {
        const zone = document.getElementById(zoneId);
        if (!zone) return;

        // Prevent default drag behaviors
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            zone.addEventListener(eventName, (e) => {
                e.preventDefault();
                e.stopPropagation();
            });
        });

        // Highlight drop zone when dragging over it
        ['dragenter', 'dragover'].forEach(eventName => {
            zone.addEventListener(eventName, () => {
                zone.classList.add('drag-over');
            });
        });

        ['dragleave', 'drop'].forEach(eventName => {
            zone.addEventListener(eventName, () => {
                zone.classList.remove('drag-over');
            });
        });

        // Handle dropped files
        zone.addEventListener('drop', (e) => {
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                onFileSelected(files[0]);
            }
        });

        // Handle click to open file picker
        zone.addEventListener('click', () => {
            const fileInput = document.getElementById('dog-photo');
            if (fileInput) {
                fileInput.click();
            }
        });
    }

    /**
     * Show upload progress indicator
     */
    showProgress() {
        // Create progress overlay if it doesn't exist
        let progressOverlay = document.getElementById('upload-progress-overlay');

        if (!progressOverlay) {
            progressOverlay = document.createElement('div');
            progressOverlay.id = 'upload-progress-overlay';
            progressOverlay.className = 'upload-progress-overlay';
            progressOverlay.innerHTML = `
                <div class="upload-progress">
                    <div class="spinner"></div>
                    <p style="margin-top: 15px; color: white;">Foto wird hochgeladen...</p>
                </div>
            `;
            document.body.appendChild(progressOverlay);
        }

        progressOverlay.style.display = 'flex';
    }

    /**
     * Hide upload progress indicator
     */
    hideProgress() {
        const progressOverlay = document.getElementById('upload-progress-overlay');
        if (progressOverlay) {
            progressOverlay.style.display = 'none';
        }
    }

    /**
     * Display current photo for editing
     */
    displayCurrentPhoto(photoUrl, containerId) {
        const container = document.getElementById(containerId);
        if (!container || !photoUrl) return;

        container.innerHTML = `
            <div class="current-photo-display">
                <img src="/uploads/${photoUrl}" alt="Current dog photo" class="current-dog-photo">
                <div class="photo-actions">
                    <button type="button" class="btn btn-small" onclick="dogPhotoManager.promptChangePhoto()">
                        Foto ändern
                    </button>
                    <button type="button" class="btn btn-danger btn-small" onclick="dogPhotoManager.promptRemovePhoto()">
                        Foto entfernen
                    </button>
                </div>
            </div>
        `;
        container.style.display = 'block';
    }

    /**
     * Hide current photo display
     */
    hideCurrentPhoto(containerId) {
        const container = document.getElementById(containerId);
        if (container) {
            container.innerHTML = '';
            container.style.display = 'none';
        }
    }

    /**
     * Prompt to change photo (shows upload UI)
     */
    promptChangePhoto() {
        // Hide current photo, show upload UI
        const currentPhotoContainer = document.getElementById('current-photo-container');
        const uploadZone = document.getElementById('photo-upload-zone');

        if (currentPhotoContainer) {
            currentPhotoContainer.style.display = 'none';
        }
        if (uploadZone) {
            uploadZone.style.display = 'block';
        }
    }

    /**
     * Prompt to remove photo
     */
    async promptRemovePhoto() {
        if (!confirm('Möchten Sie das Foto wirklich entfernen?')) {
            return;
        }

        // Note: This would require a DELETE endpoint which may not exist yet
        // For now, we'll just alert that it's not implemented
        alert('Foto-Entfernung muss noch über Backend-Endpoint implementiert werden');

        // TODO: Implement when DELETE /api/dogs/:id/photo endpoint is available
        // try {
        //     await api.removeDogPhoto(this.currentDogId);
        //     // Refresh dog data
        // } catch (error) {
        //     alert('Fehler beim Entfernen des Fotos: ' + error.message);
        // }
    }

    /**
     * Reset the photo upload form
     */
    reset() {
        this.clearPreview();
        this.selectedFile = null;
        this.currentDogId = null;
        this.hideCurrentPhoto('current-photo-container');

        // Show upload zone
        const uploadZone = document.getElementById('photo-upload-zone');
        if (uploadZone) {
            uploadZone.style.display = 'block';
        }
    }

    /**
     * Initialize for a specific dog (edit mode)
     */
    initForDog(dog) {
        this.currentDogId = dog.id;
        this.reset();

        if (dog.photo) {
            // Show current photo
            this.displayCurrentPhoto(dog.photo, 'current-photo-container');

            // Hide upload zone initially
            const uploadZone = document.getElementById('photo-upload-zone');
            if (uploadZone) {
                uploadZone.style.display = 'none';
            }
        } else {
            // No photo, show upload zone
            this.hideCurrentPhoto('current-photo-container');
            const uploadZone = document.getElementById('photo-upload-zone');
            if (uploadZone) {
                uploadZone.style.display = 'block';
            }
        }
    }
}

// Global instance
window.dogPhotoManager = new DogPhotoManager();
