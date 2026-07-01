// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Extends the default MDX component scope so that <DocBadge> is available
// in every .md / .mdx file without a per-file import.

import React from 'react';
import MDXComponents from '@theme-original/MDXComponents';
import DocBadge from '@site/src/components/DocBadge';

export default {
  ...MDXComponents,
  DocBadge,
};
