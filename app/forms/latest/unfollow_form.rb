# typed: strict
# frozen_string_literal: true

class Latest::UnfollowForm < Latest::ApplicationForm
  sig { returns(T.nilable(Profile)) }
  attr_accessor :viewer

  attribute :target_atname, :string

  validates :viewer, presence: true
  validates :target_profile, presence: true

  sig { returns(T.nilable(Profile)) }
  def target_profile
    Profile.find_by(atname: target_atname)
  end
end
