# typed: strict
# frozen_string_literal: true

class Internal::Resources::Profile < Internal::Resources::Base
  root_key :profile, :profiles

  attributes :atname, :avatar_url, :id, :name
end
