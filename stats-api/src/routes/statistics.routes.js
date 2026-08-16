import { Router } from 'express';

import { postStatistics } from '../controllers/statistics.controller.js';
import { requireAuth } from '../middleware/auth.js';

const router = Router();

router.post('/statistics', requireAuth, postStatistics);

export default router;
