# typed: strict
# frozen_string_literal: true

class Cards::ProfileCardComponent < ApplicationComponent
  renders_one :actions_menu

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile
end
