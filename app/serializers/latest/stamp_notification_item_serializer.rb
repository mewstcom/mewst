# typed: false
# frozen_string_literal: true

class Latest::StampNotificationItemSerializer < Latest::ApplicationSerializer
  one :source_profile, resource: Latest::ProfileSerializer
  one :target_post, resource: Latest::PostSerializer
end
