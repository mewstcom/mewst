# typed: strict
# frozen_string_literal: true

class Resources::Profile < Resources::Base
  root_key :profile, :profiles

  attributes :atname, :avatar_url, :name
end
