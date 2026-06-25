use std::collections::HashMap;
use crate::supervisor::ServerHandle;

pub struct ServerRegistry {
    servers: HashMap<String, ServerHandle>,
}

impl ServerRegistry {
    pub fn new() -> Self {
        Self {
            servers: HashMap::new(),
        }
    }

    pub fn insert(&mut self, id: String, handle: ServerHandle) {
        self.servers.insert(id, handle);
    }

    pub fn get(&self, id: &str) -> Option<&ServerHandle> {
        self.servers.get(id)
    }

    pub fn get_mut(&mut self, id: &str) -> Option<&mut ServerHandle> {
        self.servers.get_mut(id)
    }

    pub fn remove(&mut self, id: &str) -> Option<ServerHandle> {
        self.servers.remove(id)
    }

    pub fn list(&self) -> Vec<&ServerHandle> {
        self.servers.values().collect()
    }
}
