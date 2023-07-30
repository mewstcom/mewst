# typed: strict
# frozen_string_literal: true

class Resources::Internal::Profile < Resources::Internal::Base
  root_key :profile, :profiles

  attributes :atname, :avatar_url, :id, :name
end
