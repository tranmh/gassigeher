// Dog Photo Display Helper Functions

/**
 * Validate and sanitize a hex color code
 * @param {string} hexCode - The hex color code to validate
 * @returns {string} - Valid hex code or fallback color
 */
function sanitizeHexCode(hexCode) {
    if (!hexCode || typeof hexCode !== 'string') {
        return '#808080'; // Default gray
    }
    // Only allow valid hex color format: #RGB, #RRGGBB, or #RRGGBBAA
    const hexPattern = /^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$/;
    if (hexPattern.test(hexCode)) {
        return hexCode;
    }
    return '#808080'; // Default gray for invalid values
}

/**
 * Escape a value for safe use in HTML attributes
 * @param {any} value - The value to escape
 * @returns {string} - Safe string for HTML attributes
 */
function escapeForAttribute(value) {
    if (value === null || value === undefined) {
        return '';
    }
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');
}

/**
 * Get the photo URL for a dog, with fallback to placeholder
 * Handles both local paths (prepends /uploads/) and S3 URLs (uses directly)
 * @param {Object} dog - Dog object with photo and color fields
 * @param {boolean} useThumbnail - Whether to use thumbnail (default: false)
 * @returns {string} - Photo URL or placeholder URL
 */
function getDogPhotoUrl(dog, useThumbnail = false) {
    if (dog.photo) {
        // Use thumbnail if available and requested, otherwise use full photo
        const photoField = useThumbnail && dog.photo_thumbnail
            ? dog.photo_thumbnail
            : dog.photo;

        // Check if it's already a full URL (S3 storage) - use directly without prepending /uploads/
        if (photoField.startsWith('http://') || photoField.startsWith('https://')) {
            return photoField;
        }

        // Local storage - prepend /uploads/
        return `/uploads/${photoField}`;
    }

    return '/assets/images/placeholders/dog-placeholder.svg';
}

/**
 * Get the URL for any photo path, handling both local paths and S3 URLs
 * @param {string} photoPath - The photo path from the database
 * @param {string} fallback - Optional fallback URL if path is empty
 * @returns {string} - Photo URL ready for use in src attribute
 */
function getPhotoUrl(photoPath, fallback = '') {
    if (!photoPath) {
        return fallback;
    }

    // Check if it's already a full URL (S3 storage) - use directly
    if (photoPath.startsWith('http://') || photoPath.startsWith('https://')) {
        return photoPath;
    }

    // Local storage - prepend /uploads/
    return `/uploads/${photoPath}`;
}

// Make getPhotoUrl available globally for inline scripts in HTML files
if (typeof window !== 'undefined') {
    window.getPhotoUrl = getPhotoUrl;
}

/**
 * Get alt text for dog photo
 * @param {Object} dog - Dog object
 * @returns {string} - Alt text for image
 */
function getDogPhotoAlt(dog) {
    const safeName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dog.name) : dog.name;
    const safeBreed = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dog.breed) : dog.breed;

    if (dog.photo) {
        return `${safeName} (${safeBreed})`;
    }
    return `Kein Foto für ${safeName}`;
}

/**
 * Generate HTML for dog photo img tag with skeleton loader and fade-in
 * @param {Object} dog - Dog object
 * @param {boolean} useThumbnail - Whether to use thumbnail (default: false)
 * @param {string} className - CSS class for img element (default: 'dog-card-image')
 * @param {boolean} lazyLoad - Whether to use lazy loading (default: true)
 * @param {boolean} withSkeleton - Whether to wrap in skeleton loader (default: true)
 * @returns {string} - HTML string for img element (or container with skeleton)
 */
function getDogPhotoHtml(dog, useThumbnail = false, className = 'dog-card-image', lazyLoad = true, withSkeleton = true) {
    const photoUrl = getDogPhotoUrl(dog, useThumbnail);
    const altText = getDogPhotoAlt(dog);
    const loadingAttr = lazyLoad ? ' loading="lazy"' : '';
    // Sanitize dog.id to prevent XSS - use only alphanumeric and underscore
    const safeId = String(dog.id || Math.random().toString(36).substring(2, 11)).replace(/[^a-zA-Z0-9_-]/g, '');
    const uniqueId = `dog-img-${safeId}`;

    // For SVG placeholders, no skeleton needed
    const isSvgPlaceholder = photoUrl.includes('.svg');

    if (withSkeleton && !isSvgPlaceholder) {
        // Use data attribute instead of inline onload handler to prevent XSS
        return `<div class="dog-card-image-container" id="container-${uniqueId}">
                    <img src="${photoUrl}"
                         alt="${altText}"
                         class="${className} dog-photo-with-loader"
                         id="${uniqueId}"
                         data-image-id="${uniqueId}"
                         ${loadingAttr}>
                </div>`;
    }

    return `<img src="${photoUrl}" alt="${altText}" class="${className}"${loadingAttr}>`;
}

/**
 * Generate responsive picture element with thumbnail for mobile
 * @param {Object} dog - Dog object
 * @param {string} className - CSS class for img element (default: 'dog-card-image')
 * @param {boolean} lazyLoad - Whether to use lazy loading (default: true)
 * @returns {string} - HTML string for picture element
 */
function getDogPhotoResponsive(dog, className = 'dog-card-image', lazyLoad = true) {
    const fullUrl = getDogPhotoUrl(dog, false);
    const thumbUrl = getDogPhotoUrl(dog, true);
    const altText = getDogPhotoAlt(dog);
    const loadingAttr = lazyLoad ? ' loading="lazy"' : '';

    // If we have a thumbnail and it's different from full, use picture element
    if (dog.photo && dog.photo_thumbnail && dog.photo !== dog.photo_thumbnail) {
        return `
            <picture>
                <source media="(max-width: 768px)" srcset="${thumbUrl}">
                <img src="${fullUrl}" alt="${altText}" class="${className}"${loadingAttr}>
            </picture>
        `;
    }

    // Otherwise just use regular img
    return `<img src="${fullUrl}" alt="${altText}" class="${className}"${loadingAttr}>`;
}

/**
 * Update img element src with dog photo
 * @param {HTMLImageElement} imgElement - Image element to update
 * @param {Object} dog - Dog object
 * @param {boolean} useThumbnail - Whether to use thumbnail (default: false)
 */
function setDogPhotoSrc(imgElement, dog, useThumbnail = false) {
    if (!imgElement) return;

    const photoUrl = getDogPhotoUrl(dog, useThumbnail);
    const altText = getDogPhotoAlt(dog);

    imgElement.src = photoUrl;
    imgElement.alt = altText;
}

/**
 * Get placeholder URL for dogs without photos
 * @returns {string} - Placeholder URL
 */
function getPlaceholderUrl() {
    return '/assets/images/placeholders/dog-placeholder.svg';
}

/**
 * Handle image load event - adds fade-in effect and removes skeleton
 * @param {string} imageId - ID of the image element
 */
function handleImageLoad(imageId) {
    // Validate imageId to only contain safe characters
    if (typeof imageId !== 'string' || !/^[a-zA-Z0-9_-]+$/.test(imageId)) {
        console.warn('Invalid image ID provided to handleImageLoad');
        return;
    }

    const img = document.getElementById(imageId);
    const container = document.getElementById(`container-${imageId}`);

    if (img) {
        // Add loaded class for fade-in effect
        img.classList.add('loaded');

        // Check if image loaded from cache (instant load)
        if (img.complete && img.naturalHeight !== 0) {
            img.classList.add('no-animation');
        }
    }

    if (container) {
        // Remove skeleton animation
        container.classList.add('loaded');
    }
}

/**
 * Initialize dog photo load handlers using event delegation
 * Call this after inserting dog photo HTML into the DOM
 * @param {HTMLElement} container - Optional container to scope the initialization (defaults to document)
 */
function initDogPhotoLoadHandlers(container = document) {
    const images = container.querySelectorAll('img.dog-photo-with-loader');

    images.forEach(img => {
        // Skip if already initialized
        if (img.dataset.loadHandlerAttached) return;
        img.dataset.loadHandlerAttached = 'true';

        const imageId = img.dataset.imageId;
        if (!imageId) return;

        // Handle already loaded images (from cache)
        if (img.complete && img.naturalHeight !== 0) {
            handleImageLoad(imageId);
        } else {
            // Attach load handler
            img.addEventListener('load', function() {
                handleImageLoad(imageId);
            });

            // Handle errors gracefully
            img.addEventListener('error', function() {
                const imgContainer = document.getElementById(`container-${imageId}`);
                if (imgContainer) {
                    imgContainer.classList.add('loaded');
                }
            });
        }
    });
}

// Set up global delegated load event handler for dog photos
// This catches load events that fire before initDogPhotoLoadHandlers is called
(function() {
    if (typeof document !== 'undefined') {
        document.addEventListener('load', function(event) {
            const img = event.target;
            if (img.tagName === 'IMG' && img.classList.contains('dog-photo-with-loader')) {
                const imageId = img.dataset.imageId;
                if (imageId && !img.dataset.loadHandlerAttached) {
                    handleImageLoad(imageId);
                }
            }
        }, true); // Use capture phase to catch events early
    }
})();

/**
 * Preload critical images (first N dogs on page)
 * @param {Array} dogs - Array of dog objects
 * @param {number} count - Number of images to preload (default: 3)
 */
function preloadCriticalDogImages(dogs, count = 3) {
    if (!dogs || dogs.length === 0) return;

    const dogsToPreload = dogs.slice(0, count);

    dogsToPreload.forEach(dog => {
        if (dog.photo) {
            const link = document.createElement('link');
            link.rel = 'preload';
            link.as = 'image';
            link.href = getDogPhotoUrl(dog, false);
            document.head.appendChild(link);
        }
    });
}

/**
 * Generate HTML for calendar dog cell with photo
 * @param {Object} dog - Dog object (may include embedded color object)
 * @param {Object} color - Color object from color_categories (optional, uses dog.color if not provided)
 * @returns {string} - HTML for calendar dog name cell
 */
function getCalendarDogCell(dog, color) {
    const photoUrl = getDogPhotoUrl(dog, true); // Use thumbnail
    const altText = getDogPhotoAlt(dog);
    const safeDogName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dog.name) : dog.name;

    // Use color parameter or embedded dog.color
    const dogColor = color || dog.color;

    // Display color badge if color is available
    if (dogColor && dogColor.hex_code) {
        const patternIcons = {
            'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
            'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
            'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
            'sun': '☀', 'ring': '○', 'target': '◎'
        };
        const icon = patternIcons[dogColor.pattern_icon] || '●';
        const safeColorName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dogColor.name) : dogColor.name;
        // Sanitize hex_code to prevent CSS injection
        const safeHexCode = sanitizeHexCode(dogColor.hex_code);

        return `<div class="calendar-dog-name-cell">
            <img src="${photoUrl}" alt="${altText}" class="calendar-dog-photo" loading="lazy">
            <div>
                <div style="font-weight: 700; font-size: 1rem; color: var(--text-dark);">${safeDogName}</div>
                <span style="display: inline-flex; align-items: center; gap: 3px; font-size: 0.7rem; padding: 2px 8px; background: ${safeHexCode}20; border: 1px solid ${safeHexCode}; color: ${safeHexCode}; border-radius: 4px; margin-top: 4px;">
                    ${icon} ${safeColorName}
                </span>
            </div>
        </div>`;
    }

    // Fallback for dogs without a color assigned
    return `<div class="calendar-dog-name-cell">
        <img src="${photoUrl}" alt="${altText}" class="calendar-dog-photo" loading="lazy">
        <div>
            <div style="font-weight: 700; font-size: 1rem; color: var(--text-dark);">${safeDogName}</div>
            <span style="display: inline-block; font-size: 0.7rem; padding: 2px 8px; background: #e0e0e0; color: #666; border-radius: 4px; margin-top: 4px;">
                Keine Kategorie
            </span>
        </div>
    </div>`;
}
