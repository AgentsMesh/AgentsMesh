import { invoke } from "./invoke";
import type { IBindingService } from "@agentsmesh/service-interface";

export class ElectronBindingService implements IBindingService {
  // Connect-RPC: proto.binding.v1.BindingService. Binary wire — every
  // method forwards a Uint8Array request to the matching napi handler
  // (commands/binding.rs) and gets a Uint8Array response back. Callers
  // wrap with @bufbuild/protobuf .toBinary() / .fromBinary().

  async requestBindingConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingRequestBindingConnect", request);
  }

  async acceptBindingConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingAcceptBindingConnect", request);
  }

  async rejectBindingConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingRejectBindingConnect", request);
  }

  async unbindConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingUnbindConnect", request);
  }

  async requestScopesConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingRequestScopesConnect", request);
  }

  async approveScopesConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingApproveScopesConnect", request);
  }

  async listBindingsConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingListBindingsConnect", request);
  }

  async getPendingBindingsConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingGetPendingBindingsConnect", request);
  }

  async getBoundPodsConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingGetBoundPodsConnect", request);
  }

  async checkBindingConnect(request: Uint8Array): Promise<Uint8Array> {
    return invoke<Uint8Array>("bindingCheckBindingConnect", request);
  }
}
