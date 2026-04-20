import { Injectable } from '@nestjs/common';
import { readFileSync } from 'fs';

@Injectable()
export class AppService {
    getNetwork(): any {
        try {
            const network = readFileSync(`${process.env.VOLUME_PATH}network.json`, 'utf8');
            return JSON.parse(network);
        } catch (error) {
            console.error('Error reading network.json:', error);
            return { error: 'Failed to read network.json' };
        }
    }
}
