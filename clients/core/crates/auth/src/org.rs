use agentsmesh_types::proto_org_v1 as org_proto;
use agentsmesh_types::Organization;

use crate::connect::connect_call;
use crate::error::AuthError;
use crate::manager::AuthManager;

fn org_from_proto(o: org_proto::Organization) -> Organization {
    // proto.org.v1.Organization carries 9 fields; AuthState only needs
    // the 7 the serde DTO already exposes. subscription_plan /
    // subscription_status arrive as required strings on the wire — they
    // can be empty when the org was created before billing migrations
    // ran, so we keep `Option` semantics by promoting empty → None.
    let promote = |s: String| if s.is_empty() { None } else { Some(s) };
    Organization {
        id: o.id,
        name: o.name,
        slug: o.slug,
        role: o.role,
        logo_url: o.logo_url,
        subscription_plan: promote(o.subscription_plan),
        subscription_status: promote(o.subscription_status),
    }
}

impl AuthManager {
    pub async fn fetch_organizations(&self) -> Result<Vec<Organization>, AuthError> {
        let auth = self.bearer_header()?;
        let resp: org_proto::ListMyOrgsResponse = connect_call(
            self,
            "/proto.org.v1.OrgService/ListMyOrgs",
            &org_proto::ListMyOrgsRequest {},
            Some(&auth),
        )
        .await?;

        let orgs: Vec<Organization> = resp.items.into_iter().map(org_from_proto).collect();
        self.replace_organizations(orgs.clone());
        Ok(orgs)
    }

    pub fn get_organizations(&self) -> Vec<Organization> {
        self.read_state().organizations.clone()
    }

    pub fn get_current_org(&self) -> Option<Organization> {
        self.read_state().current_org.clone()
    }

    pub fn switch_org(&self, slug: &str) -> Result<(), AuthError> {
        let org = self
            .read_state()
            .organizations
            .iter()
            .find(|o| o.slug == slug)
            .cloned()
            .ok_or_else(|| {
                AuthError::InvalidResponse(format!("organization '{slug}' not found"))
            })?;
        self.set_current_org(Some(org));
        Ok(())
    }
}
