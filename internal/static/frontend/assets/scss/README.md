# SASS Architecture - Gassigeher

This directory contains the SASS source files for the Gassigeher project, organized following the **7-1 Pattern** and **ITCSS (Inverted Triangle CSS)** principles for maximum maintainability and scalability.

## 📁 Directory Structure

```
scss/
├── abstracts/          # Design tokens, no CSS output
│   ├── _variables.scss # Colors, spacing, typography, etc.
│   ├── _mixins.scss    # Reusable SASS mixins
│   └── _index.scss     # Exports all abstracts
│
├── base/               # Foundational styles
│   ├── _reset.scss     # CSS reset and base elements
│   ├── _typography.scss # Heading and text styles
│   ├── _utilities.scss # Utility classes (.mt-1, .text-center, etc.)
│   └── _index.scss     # Exports all base styles
│
├── components/         # Reusable UI components
│   ├── _buttons.scss   # Button styles and variants
│   ├── _forms.scss     # Input fields, labels, validation
│   ├── _cards.scss     # Card containers
│   ├── _alerts.scss    # Alert messages
│   ├── _spinner.scss   # Loading spinners
│   ├── _skeleton.scss  # Skeleton loaders
│   ├── _badges.scss    # Category and featured badges
│   ├── _dog-cards.scss # Dog-specific card styles
│   ├── _photo-upload.scss # Photo upload zone
│   ├── _filters.scss   # Filter bar
│   ├── _features.scss  # Feature grid
│   └── _index.scss     # Exports all components
│
├── layout/             # Major page sections
│   ├── _containers.scss # Container widths
│   ├── _header.scss    # Site header
│   ├── _navigation.scss # Main navigation and dropdowns
│   ├── _footer.scss    # Site footer
│   ├── _hero.scss      # Hero banner
│   └── _index.scss     # Exports all layout
│
├── pages/              # Page-specific styles
│   └── _index.scss     # (Currently empty - all styles are component-based)
│
└── main.scss           # Main entry point, imports everything
```

## 🎯 Design Principles

### 1. Single Responsibility Principle
Each SASS file has one clear purpose:
- **Abstracts**: Define design tokens (variables, mixins)
- **Base**: Style HTML elements directly
- **Components**: Style reusable UI patterns
- **Layout**: Structure major page sections
- **Pages**: Override styles for specific pages

### 2. DRY (Don't Repeat Yourself)
- All colors, spacing, fonts are defined once in `_variables.scss`
- Common patterns extracted to mixins in `_mixins.scss`
- Use `@include` to apply mixins, not copy-paste CSS

### 3. Mobile-First Responsive Design
```scss
// Default styles for mobile
.button { padding: 10px; }

// Desktop overrides
@include desktop {
  .button { padding: 15px; }
}
```

### 4. BEM Naming Convention
```scss
.dog-card { }              // Block
.dog-card__image { }       // Element
.dog-card--featured { }    // Modifier
```

## 🔧 Usage

### Development Workflow

**Start watching for changes:**
```bash
npm run sass:watch
```
This automatically recompiles SASS whenever you save a file.

**Compile once:**
```bash
npm run sass
```

**Production build (minified):**
```bash
npm run sass:prod
```

### Adding New Styles

**1. Choose the right location:**
- **Variables/Mixins**: Add to `abstracts/`
- **New component**: Create new file in `components/`
- **Layout change**: Edit existing file in `layout/`
- **Global utility**: Add to `base/_utilities.scss`

**2. Follow the pattern:**

```scss
// components/_new-component.scss
@use '../abstracts' as *;

.new-component {
  background: $card-bg;
  padding: $spacing-md;
  @include shadow-md;
  @include transition;

  &:hover {
    @include shadow-lg;
  }
}
```

**3. Import in the index file:**

```scss
// components/_index.scss
@forward 'buttons';
@forward 'forms';
@forward 'new-component'; // Add your new component
```

### Modifying Colors

Edit `abstracts/_variables.scss`:

```scss
// Change primary color
$primary-green: #82b965;  // Edit this value

// All components using $primary-green automatically update!
```

Then recompile: `npm run sass`

### Creating New Mixins

Add to `abstracts/_mixins.scss`:

```scss
@mixin custom-pattern {
  // Your reusable pattern
  border: 2px solid $primary-green;
  @include transition;

  &:hover {
    border-color: $secondary-green;
  }
}
```

Use it anywhere:

```scss
.my-element {
  @include custom-pattern;
}
```

## 🎨 Available Design Tokens

### Colors
```scss
$primary-green      // #82b965
$secondary-green    // #6fa050
$accent-orange      // #ff8c42
$accent-blue        // #4a90e2
$text-dark          // #2c3e34
$error-red          // #e74c3c
// ... see _variables.scss for full list
```

### Spacing
```scss
$spacing-xs   // 0.44rem (7px)
$spacing-sm   // 0.88rem (14px)
$spacing-md   // 1.32rem (21px)
$spacing-lg   // 1.76rem (28px)
$spacing-xl   // 2.64rem (42px)
```

### Mixins
```scss
@include mobile { }           // Mobile breakpoint
@include desktop { }          // Desktop breakpoint
@include flex-center { }      // Flexbox centering
@include gradient-primary { } // Green gradient
@include shadow-md { }        // Medium shadow
@include transition { }       // Smooth transitions
@include hover-lift { }       // Lift on hover
// ... see _mixins.scss for full list
```

## 📝 Best Practices

### ✅ DO

```scss
// Use variables for all values
.button {
  padding: $spacing-md;
  color: $primary-green;
  border-radius: $border-radius;
}

// Use mixins for common patterns
.card {
  @include card-base;
  @include card-hover;
}

// Nest selectors logically
.nav-dropdown {
  position: relative;

  &:hover .nav-dropdown-menu {
    display: block;
  }
}
```

### ❌ DON'T

```scss
// Don't use hardcoded values
.button {
  padding: 12px 30px;  // ❌ Use $btn-padding instead
  color: #82b965;      // ❌ Use $primary-green instead
}

// Don't repeat the same CSS
.card1 { box-shadow: 0 3px 12px rgba(0,0,0,0.08); }
.card2 { box-shadow: 0 3px 12px rgba(0,0,0,0.08); }
// ❌ Create a mixin or variable instead

// Don't nest too deeply (max 3 levels)
.nav {
  ul {
    li {
      a {
        span { }  // ❌ Too deep!
      }
    }
  }
}
```

## 🔍 Debugging SASS Compilation

**Error: "Undefined variable"**
- Make sure you have `@use '../abstracts' as *;` at the top of your file

**Error: "Undefined mixin"**
- Check that the mixin is defined in `_mixins.scss`
- Verify you're importing abstracts: `@use '../abstracts' as *;`

**Styles not updating?**
- Check that `npm run sass:watch` is running
- Hard refresh browser (Ctrl+Shift+R / Cmd+Shift+R)
- Check browser console for CSS loading errors

**Production CSS too large?**
- Use `npm run sass:prod` for minified output
- Remove unused components from imports

## 📚 Further Reading

- [SASS Documentation](https://sass-lang.com/documentation)
- [7-1 Pattern](https://sass-guidelin.es/#the-7-1-pattern)
- [BEM Methodology](http://getbem.com/)
- [ITCSS Architecture](https://www.xfive.co/blog/itcss-scalable-maintainable-css-architecture/)

## 🤝 Contributing

When adding new styles:

1. **Check existing patterns first** - Don't duplicate!
2. **Use design tokens** - Variables and mixins, not hardcoded values
3. **Test responsiveness** - Check mobile, tablet, desktop
4. **Follow naming conventions** - BEM for components
5. **Document complex patterns** - Add comments for future developers
6. **Keep it modular** - One component per file

---

**Questions?** Check the main [CLAUDE.md](../../../../CLAUDE.md) for complete project documentation.
