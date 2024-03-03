# typed: strict
# frozen_string_literal: true

class PostEntity < ApplicationEntity
  sig { returns(T::Mewst::DatabaseId) }
  attr_reader :id

  sig { returns(String) }
  attr_reader :content

  sig { returns(T::Boolean) }
  attr_reader :viewer_has_stamped

  sig { returns(Integer) }
  attr_reader :stamps_count

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :published_at

  sig { returns(ProfileEntity) }
  attr_reader :profile_entity

  sig do
    params(
      id: T::Mewst::DatabaseId,
      content: String,
      viewer_has_stamped: T::Boolean,
      stamps_count: Integer,
      published_at: ActiveSupport::TimeWithZone,
      profile_entity: ProfileEntity
    )
  end
  def initialize(id:, content:, viewer_has_stamped:, stamps_count:, published_at:, profile_entity:)
    @id = id
    @content = content
    @viewer_has_stamped = viewer_has_stamped
    @stamps_count = stamps_count
    @published_at = published_at
    @profile_entity = profile_entity
  end
end
