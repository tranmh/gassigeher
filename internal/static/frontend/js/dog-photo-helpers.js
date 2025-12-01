// Dog Photo Display Helper Functions

/**
 * Get the photo URL for a dog, with fallback to placeholder
 * @param {Object} dog - Dog object with photo and category fields
 * @param {boolean} useThumbnail - Whether to use thumbnail (default: false)
 * @param {boolean} useCategoryPlaceholder - Whether to use category-specific placeholder (default: true)
 * @returns {string} - Photo URL or placeholder URL
 */
function getDogPhotoUrl(dog, useThumbnail = false, useCategoryPlaceholder = true) {
    if (dog.photo) {
        // Use thumbnail if available and requested, otherwise use full photo
        const photoField = useThumbnail && dog.photo_thumbnail
            ? dog.photo_thumbnail
            : dog.photo;
        return `/uploads/${photoField}`;
    }

    // Return category-specific placeholder or generic placeholder
    if (useCategoryPlaceholder && dog.category) {
        const category = dog.category.toLowerCase();
        if (['green', 'blue', 'orange'].includes(category)) {
            return `/assets/images/placeholders/dog-placeholder-${category}.svg`;
        }
    }

    return '/assets/images/placeholders/dog-placeholder.svg';
}

/**
 * Get alt text for dog photo
 * @param {Object} dog - Dog object
 * @returns {string} - Alt text for image
 */
function getDogPhotoAlt(dog) {
    if (dog.photo) {
        return `${dog.name} (${dog.breed})`;
    }
    return `Kein Foto für ${dog.name}`;
}

/**
 * Generate HTML for dog photo img tag with skeleton loader and fade-in
 * @param {Object} dog - Dog object
 * @param {boolean} useThumbnail - Whether to use thumbnail (default: false)
 * @param {string} className - CSS class for img element (default: 'dog-card-image')
 * @param {boolean} lazyLoad - Whether to use lazy loading (default: true)
 * @param {boolean} useCategoryPlaceholder - Whether to use category-specific placeholder (default: true)
 * @param {boolean} withSkeleton - Whether to wrap in skeleton loader (default: true)
 * @returns {string} - HTML string for img element (or container with skeleton)
 */
function getDogPhotoHtml(dog, useThumbnail = false, className = 'dog-card-image', lazyLoad = true, useCategoryPlaceholder = true, withSkeleton = true) {
    const photoUrl = getDogPhotoUrl(dog, useThumbnail, useCategoryPlaceholder);
    const altText = getDogPhotoAlt(dog);
    const loadingAttr = lazyLoad ? ' loading="lazy"' : '';
    const uniqueId = `dog-img-${dog.id || Math.random().toString(36).substr(2, 9)}`;

    // For SVG placeholders, no skeleton needed
    const isSvgPlaceholder = photoUrl.includes('.svg');

    if (withSkeleton && !isSvgPlaceholder) {
        return `<div class="dog-card-image-container" id="container-${uniqueId}">
                    <img src="${photoUrl}"
                         alt="${altText}"
                         class="${className}"
                         id="${uniqueId}"
                         ${loadingAttr}
                         onload="handleImageLoad('${uniqueId}')">
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
 * Get placeholder URL for a specific category
 * @param {string} category - 'green', 'blue', or 'orange'
 * @returns {string} - Placeholder URL
 */
function getPlaceholderUrl(category = null) {
    if (category) {
        const cat = category.toLowerCase();
        if (['green', 'blue', 'orange'].includes(cat)) {
            return `/assets/images/placeholders/dog-placeholder-${cat}.svg`;
        }
    }
    return '/assets/images/placeholders/dog-placeholder.svg';
}

/**
 * Handle image load event - adds fade-in effect and removes skeleton
 * @param {string} imageId - ID of the image element
 */
function handleImageLoad(imageId) {
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
 * @param {Object} dog - Dog object
 * @returns {string} - HTML for calendar dog name cell
 */
function getCalendarDogCell(dog) {
    const photoUrl = getDogPhotoUrl(dog, true, true); // Use thumbnail
    const altText = getDogPhotoAlt(dog);

    const categoryEmoji = {
        'green': '🟢',
        'blue': '🔵',
        'orange': '🟠'
    }[dog.category] || '⚪';

    const categoryColor = {
        'green': '#c3e6cb',
        'blue': '#bee5eb',
        'orange': '#ffe69c'
    }[dog.category] || '#e0e0e0';

    const categoryLabel = {
        'green': 'Grün',
        'blue': 'Blau',
        'orange': 'Orange'
    }[dog.category] || dog.category;

    return `<div class="calendar-dog-name-cell">
        <img src="${photoUrl}"
             alt="${altText}"
             class="calendar-dog-photo"
             loading="lazy">
        <div>
            <div style="font-weight: 700; font-size: 1rem; color: var(--text-dark);">${dog.name}</div>
            <span style="display: inline-block; font-size: 0.7rem; padding: 2px 8px; background: ${categoryColor}; border-radius: 4px; margin-top: 4px;">
                ${categoryEmoji} ${categoryLabel}
            </span>
        </div>
    </div>`;
}
