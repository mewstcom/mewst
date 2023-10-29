# typed: false
# frozen_string_literal: true

class Latest::FollowNotificationItemSerializer < Latest::ApplicationSerializer
  one :source_profile, resource: Latest::ProfileSerializer
end
