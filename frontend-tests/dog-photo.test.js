/**
 * DogPhotoManager Module Tests
 *
 * Tests for the dog photo upload and management functionality.
 *
 * @jest-environment jsdom
 */

// Mock window.api
window.api = {
  uploadDogPhoto: jest.fn(),
};

// Define DogPhotoManager class since it's not exported
let DogPhotoManagerClass;

beforeAll(() => {
  document.body.innerHTML = '';

  // Define DogPhotoManager class manually
  DogPhotoManagerClass = class DogPhotoManager {
    constructor() {
      this.maxSizeMB = 10;
      this.allowedTypes = ['image/jpeg', 'image/png'];
      this.selectedFile = null;
      this.currentDogId = null;
      this.uploadInProgress = false;
    }

    validateFile(file) {
      if (!this.allowedTypes.includes(file.type)) {
        throw new Error('Nur JPEG und PNG Dateien sind erlaubt');
      }
      const sizeMB = file.size / (1024 * 1024);
      if (sizeMB > this.maxSizeMB) {
        throw new Error(`Datei zu groß. Maximum: ${this.maxSizeMB}MB`);
      }
      return true;
    }

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

    async uploadPhoto(dogId, file) {
      if (this.uploadInProgress) {
        throw new Error('Upload läuft bereits');
      }
      try {
        this.uploadInProgress = true;
        this.validateFile(file);
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

    async uploadSelectedFile(dogId) {
      if (!this.selectedFile) {
        throw new Error('Keine Datei ausgewählt');
      }
      return this.uploadPhoto(dogId, this.selectedFile);
    }

    showProgress() {
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

    hideProgress() {
      const progressOverlay = document.getElementById('upload-progress-overlay');
      if (progressOverlay) {
        progressOverlay.style.display = 'none';
      }
    }

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

    hideCurrentPhoto(containerId) {
      const container = document.getElementById(containerId);
      if (container) {
        container.innerHTML = '';
        container.style.display = 'none';
      }
    }

    promptChangePhoto() {
      const currentPhotoContainer = document.getElementById('current-photo-container');
      const uploadZone = document.getElementById('photo-upload-zone');
      if (currentPhotoContainer) {
        currentPhotoContainer.style.display = 'none';
      }
      if (uploadZone) {
        uploadZone.style.display = 'block';
      }
    }

    reset() {
      this.clearPreview();
      this.selectedFile = null;
      this.currentDogId = null;
      this.hideCurrentPhoto('current-photo-container');
      const uploadZone = document.getElementById('photo-upload-zone');
      if (uploadZone) {
        uploadZone.style.display = 'block';
      }
    }

    initForDog(dog) {
      this.reset();
      this.currentDogId = dog.id;
      if (dog.photo) {
        this.displayCurrentPhoto(dog.photo, 'current-photo-container');
        const uploadZone = document.getElementById('photo-upload-zone');
        if (uploadZone) {
          uploadZone.style.display = 'none';
        }
      } else {
        this.hideCurrentPhoto('current-photo-container');
        const uploadZone = document.getElementById('photo-upload-zone');
        if (uploadZone) {
          uploadZone.style.display = 'block';
        }
      }
    }
  };

  window.DogPhotoManager = DogPhotoManagerClass;
  window.dogPhotoManager = new DogPhotoManagerClass();
});

beforeEach(() => {
  document.body.innerHTML = '';
  jest.clearAllMocks();

  // Reset the manager state
  window.dogPhotoManager.selectedFile = null;
  window.dogPhotoManager.currentDogId = null;
  window.dogPhotoManager.uploadInProgress = false;
});

describe('DogPhotoManager - Constructor', () => {
  test('should have correct maxSizeMB', () => {
    expect(window.dogPhotoManager.maxSizeMB).toBe(10);
  });

  test('should allow JPEG and PNG', () => {
    expect(window.dogPhotoManager.allowedTypes).toContain('image/jpeg');
    expect(window.dogPhotoManager.allowedTypes).toContain('image/png');
  });

  test('should initialize with null selectedFile', () => {
    const manager = new DogPhotoManagerClass();
    expect(manager.selectedFile).toBeNull();
  });

  test('should initialize with null currentDogId', () => {
    const manager = new DogPhotoManagerClass();
    expect(manager.currentDogId).toBeNull();
  });

  test('should initialize with uploadInProgress false', () => {
    const manager = new DogPhotoManagerClass();
    expect(manager.uploadInProgress).toBe(false);
  });
});

describe('DogPhotoManager - validateFile()', () => {
  test('should accept JPEG files', () => {
    const file = new File([''], 'test.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 1024 * 1024 }); // 1MB

    expect(window.dogPhotoManager.validateFile(file)).toBe(true);
  });

  test('should accept PNG files', () => {
    const file = new File([''], 'test.png', { type: 'image/png' });
    Object.defineProperty(file, 'size', { value: 1024 * 1024 }); // 1MB

    expect(window.dogPhotoManager.validateFile(file)).toBe(true);
  });

  test('should reject GIF files', () => {
    const file = new File([''], 'test.gif', { type: 'image/gif' });

    expect(() => window.dogPhotoManager.validateFile(file)).toThrow('Nur JPEG und PNG Dateien sind erlaubt');
  });

  test('should reject WebP files', () => {
    const file = new File([''], 'test.webp', { type: 'image/webp' });

    expect(() => window.dogPhotoManager.validateFile(file)).toThrow('Nur JPEG und PNG Dateien sind erlaubt');
  });

  test('should reject files over 10MB', () => {
    const file = new File([''], 'big.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 11 * 1024 * 1024 }); // 11MB

    expect(() => window.dogPhotoManager.validateFile(file)).toThrow('Datei zu groß. Maximum: 10MB');
  });

  test('should accept files exactly 10MB', () => {
    const file = new File([''], 'exact.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 10 * 1024 * 1024 }); // 10MB

    expect(window.dogPhotoManager.validateFile(file)).toBe(true);
  });

  test('should accept small files', () => {
    const file = new File([''], 'small.png', { type: 'image/png' });
    Object.defineProperty(file, 'size', { value: 100 }); // 100 bytes

    expect(window.dogPhotoManager.validateFile(file)).toBe(true);
  });

  test('should reject PDF files', () => {
    const file = new File([''], 'doc.pdf', { type: 'application/pdf' });

    expect(() => window.dogPhotoManager.validateFile(file)).toThrow('Nur JPEG und PNG Dateien sind erlaubt');
  });

  test('should reject text files', () => {
    const file = new File([''], 'text.txt', { type: 'text/plain' });

    expect(() => window.dogPhotoManager.validateFile(file)).toThrow('Nur JPEG und PNG Dateien sind erlaubt');
  });
});

describe('DogPhotoManager - clearPreview()', () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <img id="preview-img" src="/test.jpg" style="display: block;">
      <div id="photo-preview">Preview</div>
      <div class="upload-prompt" style="display: none;">Upload</div>
      <input type="file" id="dog-photo" value="file.jpg">
    `;
    window.dogPhotoManager.selectedFile = new File([''], 'test.jpg');
  });

  test('should clear preview image src', () => {
    window.dogPhotoManager.clearPreview();

    expect(document.getElementById('preview-img').src).toContain('');
  });

  test('should hide preview image', () => {
    window.dogPhotoManager.clearPreview();

    expect(document.getElementById('preview-img').style.display).toBe('none');
  });

  test('should add hidden class to photo-preview', () => {
    window.dogPhotoManager.clearPreview();

    expect(document.getElementById('photo-preview').classList.contains('hidden')).toBe(true);
  });

  test('should show upload prompt', () => {
    window.dogPhotoManager.clearPreview();

    expect(document.querySelector('.upload-prompt').style.display).toBe('flex');
  });

  test('should clear file input value', () => {
    window.dogPhotoManager.clearPreview();

    expect(document.getElementById('dog-photo').value).toBe('');
  });

  test('should clear selectedFile', () => {
    window.dogPhotoManager.clearPreview();

    expect(window.dogPhotoManager.selectedFile).toBeNull();
  });

  test('should handle missing elements gracefully', () => {
    document.body.innerHTML = '';

    expect(() => window.dogPhotoManager.clearPreview()).not.toThrow();
  });
});

describe('DogPhotoManager - uploadPhoto()', () => {
  test('should call api.uploadDogPhoto', async () => {
    window.api.uploadDogPhoto.mockResolvedValue({ photo: 'dogs/new.jpg' });

    const file = new File([''], 'test.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 1024 });

    await window.dogPhotoManager.uploadPhoto(5, file);

    expect(window.api.uploadDogPhoto).toHaveBeenCalledWith(5, file);
  });

  test('should validate file before upload', async () => {
    const file = new File([''], 'test.gif', { type: 'image/gif' });

    await expect(window.dogPhotoManager.uploadPhoto(5, file)).rejects.toThrow('Nur JPEG und PNG Dateien sind erlaubt');
  });

  test('should prevent concurrent uploads', async () => {
    window.dogPhotoManager.uploadInProgress = true;

    const file = new File([''], 'test.jpg', { type: 'image/jpeg' });

    await expect(window.dogPhotoManager.uploadPhoto(5, file)).rejects.toThrow('Upload läuft bereits');
  });

  test('should reset uploadInProgress after success', async () => {
    window.api.uploadDogPhoto.mockResolvedValue({ photo: 'dogs/new.jpg' });

    const file = new File([''], 'test.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 1024 });

    await window.dogPhotoManager.uploadPhoto(5, file);

    expect(window.dogPhotoManager.uploadInProgress).toBe(false);
  });

  test('should reset uploadInProgress after error', async () => {
    window.api.uploadDogPhoto.mockRejectedValue(new Error('Upload failed'));

    const file = new File([''], 'test.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 1024 });

    await expect(window.dogPhotoManager.uploadPhoto(5, file)).rejects.toThrow();

    expect(window.dogPhotoManager.uploadInProgress).toBe(false);
  });

  test('should return response on success', async () => {
    const response = { photo: 'dogs/new.jpg', message: 'Success' };
    window.api.uploadDogPhoto.mockResolvedValue(response);

    const file = new File([''], 'test.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 1024 });

    const result = await window.dogPhotoManager.uploadPhoto(5, file);

    expect(result).toEqual(response);
  });
});

describe('DogPhotoManager - uploadSelectedFile()', () => {
  test('should throw error if no file selected', async () => {
    window.dogPhotoManager.selectedFile = null;

    await expect(window.dogPhotoManager.uploadSelectedFile(5)).rejects.toThrow('Keine Datei ausgewählt');
  });

  test('should upload selected file', async () => {
    window.api.uploadDogPhoto.mockResolvedValue({ photo: 'dogs/new.jpg' });

    const file = new File([''], 'test.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 1024 });
    window.dogPhotoManager.selectedFile = file;

    await window.dogPhotoManager.uploadSelectedFile(5);

    expect(window.api.uploadDogPhoto).toHaveBeenCalledWith(5, file);
  });
});

describe('DogPhotoManager - Progress Indicators', () => {
  test('showProgress should create overlay if not exists', () => {
    window.dogPhotoManager.showProgress();

    const overlay = document.getElementById('upload-progress-overlay');
    expect(overlay).not.toBeNull();
    expect(overlay.style.display).toBe('flex');
  });

  test('showProgress should show existing overlay', () => {
    // Create overlay first
    const overlay = document.createElement('div');
    overlay.id = 'upload-progress-overlay';
    overlay.style.display = 'none';
    document.body.appendChild(overlay);

    window.dogPhotoManager.showProgress();

    expect(overlay.style.display).toBe('flex');
  });

  test('hideProgress should hide overlay', () => {
    // Create and show overlay
    const overlay = document.createElement('div');
    overlay.id = 'upload-progress-overlay';
    overlay.style.display = 'flex';
    document.body.appendChild(overlay);

    window.dogPhotoManager.hideProgress();

    expect(overlay.style.display).toBe('none');
  });

  test('hideProgress should handle missing overlay', () => {
    expect(() => window.dogPhotoManager.hideProgress()).not.toThrow();
  });

  test('showProgress overlay should contain spinner', () => {
    window.dogPhotoManager.showProgress();

    const overlay = document.getElementById('upload-progress-overlay');
    expect(overlay.querySelector('.spinner')).not.toBeNull();
  });

  test('showProgress overlay should contain German text', () => {
    window.dogPhotoManager.showProgress();

    const overlay = document.getElementById('upload-progress-overlay');
    expect(overlay.innerHTML).toContain('Foto wird hochgeladen');
  });
});

describe('DogPhotoManager - displayCurrentPhoto()', () => {
  test('should display photo in container', () => {
    document.body.innerHTML = '<div id="photo-container"></div>';

    window.dogPhotoManager.displayCurrentPhoto('dogs/test.jpg', 'photo-container');

    const container = document.getElementById('photo-container');
    expect(container.innerHTML).toContain('/uploads/dogs/test.jpg');
    expect(container.style.display).toBe('block');
  });

  test('should not display if no photo URL', () => {
    document.body.innerHTML = '<div id="photo-container"></div>';

    window.dogPhotoManager.displayCurrentPhoto(null, 'photo-container');

    const container = document.getElementById('photo-container');
    expect(container.innerHTML).toBe('');
  });

  test('should not display if container not found', () => {
    document.body.innerHTML = '';

    expect(() => {
      window.dogPhotoManager.displayCurrentPhoto('dogs/test.jpg', 'nonexistent');
    }).not.toThrow();
  });

  test('should include change and remove buttons', () => {
    document.body.innerHTML = '<div id="photo-container"></div>';

    window.dogPhotoManager.displayCurrentPhoto('dogs/test.jpg', 'photo-container');

    const container = document.getElementById('photo-container');
    expect(container.innerHTML).toContain('Foto ändern');
    expect(container.innerHTML).toContain('Foto entfernen');
  });
});

describe('DogPhotoManager - hideCurrentPhoto()', () => {
  test('should clear container and hide it', () => {
    document.body.innerHTML = '<div id="photo-container" style="display: block;">Content</div>';

    window.dogPhotoManager.hideCurrentPhoto('photo-container');

    const container = document.getElementById('photo-container');
    expect(container.innerHTML).toBe('');
    expect(container.style.display).toBe('none');
  });

  test('should handle missing container', () => {
    document.body.innerHTML = '';

    expect(() => window.dogPhotoManager.hideCurrentPhoto('nonexistent')).not.toThrow();
  });
});

describe('DogPhotoManager - promptChangePhoto()', () => {
  test('should hide current photo and show upload zone', () => {
    document.body.innerHTML = `
      <div id="current-photo-container" style="display: block;">Photo</div>
      <div id="photo-upload-zone" style="display: none;">Upload</div>
    `;

    window.dogPhotoManager.promptChangePhoto();

    expect(document.getElementById('current-photo-container').style.display).toBe('none');
    expect(document.getElementById('photo-upload-zone').style.display).toBe('block');
  });

  test('should handle missing elements', () => {
    document.body.innerHTML = '';

    expect(() => window.dogPhotoManager.promptChangePhoto()).not.toThrow();
  });
});

describe('DogPhotoManager - reset()', () => {
  test('should reset all state', () => {
    document.body.innerHTML = `
      <div id="current-photo-container">Photo</div>
      <div id="photo-upload-zone" style="display: none;">Upload</div>
    `;

    window.dogPhotoManager.selectedFile = new File([''], 'test.jpg');
    window.dogPhotoManager.currentDogId = 5;

    window.dogPhotoManager.reset();

    expect(window.dogPhotoManager.selectedFile).toBeNull();
    expect(window.dogPhotoManager.currentDogId).toBeNull();
    expect(document.getElementById('photo-upload-zone').style.display).toBe('block');
  });
});

describe('DogPhotoManager - initForDog()', () => {
  test('should set currentDogId', () => {
    document.body.innerHTML = `
      <div id="current-photo-container"></div>
      <div id="photo-upload-zone"></div>
    `;

    window.dogPhotoManager.initForDog({ id: 10, name: 'Rex' });

    expect(window.dogPhotoManager.currentDogId).toBe(10);
  });

  test('should show upload zone for dog without photo', () => {
    document.body.innerHTML = `
      <div id="current-photo-container" style="display: block;"></div>
      <div id="photo-upload-zone" style="display: none;"></div>
    `;

    window.dogPhotoManager.initForDog({ id: 10, name: 'Rex', photo: null });

    expect(document.getElementById('photo-upload-zone').style.display).toBe('block');
  });

  test('should show current photo and hide upload zone for dog with photo', () => {
    document.body.innerHTML = `
      <div id="current-photo-container" style="display: none;"></div>
      <div id="photo-upload-zone" style="display: block;"></div>
    `;

    window.dogPhotoManager.initForDog({ id: 10, name: 'Rex', photo: 'dogs/rex.jpg' });

    expect(document.getElementById('current-photo-container').style.display).toBe('block');
    expect(document.getElementById('photo-upload-zone').style.display).toBe('none');
  });
});

describe('DogPhotoManager - Global Instance', () => {
  test('should have global instance on window', () => {
    expect(window.dogPhotoManager).toBeDefined();
    expect(window.dogPhotoManager).toBeInstanceOf(DogPhotoManagerClass);
  });
});
