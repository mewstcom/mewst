# typed: strict
# frozen_string_literal: true

class V1::SuggestedFollowForm < V1::ApplicationForm
  sig { returns(T.nilable(Actor)) }
  attr_accessor :viewer

  attribute :target_atname, :string

  validates :viewer, presence: true
  validates :target_profile, presence: true

  sig { returns(T.nilable(Profile)) }
  def target_profile
    Profile.kept.find_by(atname: target_atname)
  end
end
