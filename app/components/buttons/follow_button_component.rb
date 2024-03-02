# typed: strict
# frozen_string_literal: true

class Buttons::FollowButtonComponent < ApplicationComponent
  sig { params(target_profile_entity: ProfileEntity).void }
  def initialize(target_profile_entity:)
    @target_profile_entity = target_profile_entity
  end

  sig { returns(ProfileEntity) }
  attr_reader :target_profile_entity
  private :target_profile_entity
end
