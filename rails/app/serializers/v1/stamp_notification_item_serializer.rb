# typed: false
# frozen_string_literal: true

class V1::StampNotificationItemSerializer < V1::ApplicationSerializer
  one :source_profile, resource: V1::ProfileSerializer
  one :target_post, resource: V1::PostSerializer
end
