# typed: false
# frozen_string_literal: true

class Latest::StampNotificationItemSerializer < Latest::ApplicationSerializer
  one :profile, resource: Latest::ProfileSerializer
  one :post, resource: Latest::PostSerializer
end
