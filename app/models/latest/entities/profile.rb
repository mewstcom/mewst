# typed: strict
# frozen_string_literal: true

class Latest::Entities::Profile < Latest::Entities::Base
  delegate :atname, :avatar_url, :name, to: :profile

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile
end
