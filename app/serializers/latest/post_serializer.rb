# typed: false
# frozen_string_literal: true

class Latest::PostSerializer < Latest::ApplicationSerializer
  root_key :post, :posts

  attributes :id, :content, :published_at, :stamps_count, :viewer_has_stamped

  one :profile, resource: Latest::ProfileSerializer
  one :via, resource: Latest::ViaSerializer
end
