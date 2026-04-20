export default class Config {
    /**
     * @param {pino.Logger} logger logger instance
     */
    constructor(logger) {
        this.logger = logger;
        
        this.ENVIRONMENT_ENTRY = 'environment';
        this.NODES_FIELD = 'nodes';

        this.BOOTSTRAP_ENTRY = 'bootstrap';
        this.BOOTSTRAP_PORT = 'port';
    }

    checkConfig(configuration) {
        if (!Object.hasOwn(configuration, this.ENVIRONMENT_ENTRY)) {
            this.logger.error(`Config: No '${this.ENVIRONMENT_ENTRY}'`);
            return false;
        }

        if (!Object.hasOwn(configuration, this.BOOTSTRAP_ENTRY)) {
            this.logger.error(`Config: No '${this.BOOTSTRAP_ENTRY}'`);
            return false;
        }

        // Environment
        if (!Object.hasOwn(configuration[this.ENVIRONMENT_ENTRY], this.NODES_FIELD)) {
            this.logger.error(`Config: No '${this.NODES_FIELD}'`);
            return false;
        }

        // Bootstrap
        if (!Object.hasOwn(configuration[this.BOOTSTRAP_ENTRY], this.BOOTSTRAP_PORT)) {
            this.logger.error(`Config: No '${this.BOOTSTRAP_PORT}'`);
            return false;
        }

        return true;
    }
}
