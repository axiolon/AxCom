// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

import React from 'react';
import styles from './styles.module.css';

type Status = 'stable' | 'under-review' | 'experimental' | 'beta' | 'deprecated';

interface DocBadgeProps {
  /** Current review/stability status of the document. */
  status?: Status;
  /** The release version this document was introduced with (e.g. "v0.1.0-alpha"). */
  version?: string;
  /** The release version this document was last reviewed against. */
  reviewedAt?: string;
}

const STATUS_LABELS: Record<Status, string> = {
  'stable':       'Stable',
  'under-review': 'Under Review',
  'experimental': 'Experimental',
  'beta':         'Beta',
  'deprecated':   'Deprecated',
};

export default function DocBadge({ status, version, reviewedAt }: DocBadgeProps): JSX.Element {
  return (
    <div className={styles.badgeRow}>
      {status && (
        <span className={`${styles.badge} ${styles[status.replace('-', '_')]}`}>
          {STATUS_LABELS[status]}
        </span>
      )}
      {version && (
        <span className={`${styles.badge} ${styles.version}`}>
          {version}
        </span>
      )}
      {reviewedAt && (
        <span className={`${styles.badge} ${styles.reviewed}`}>
          Reviewed {reviewedAt}
        </span>
      )}
    </div>
  );
}
