# typed: false
# frozen_string_literal: true

class V1::PostSerializer < V1::ApplicationSerializer
  root_key :post, :posts

  attributes :id, :content, :published_at, :stamps_count, :viewer_has_stamped

  one :profile, resource: V1::ProfileSerializer
  one :via, resource: V1::ViaSerializer
end
