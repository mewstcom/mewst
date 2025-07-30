# typed: strict
# frozen_string_literal: true

class Cards::ProfileCardComponent < ApplicationComponent
  renders_one :actions_menu

  sig { params(profile: ProfileRecord).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { returns(ProfileRecord) }
  attr_reader :profile
  private :profile
end
