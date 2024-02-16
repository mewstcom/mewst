# typed: strict
# frozen_string_literal: true

module ProfilesHelper
  extend T::Sig

  sig { params(profile: Profile).returns(String) }
  def name_with_atname(profile:)
    profile.name.present? ? "#{profile.name} (@#{profile.atname})" : "@#{profile.atname}"
  end
end
